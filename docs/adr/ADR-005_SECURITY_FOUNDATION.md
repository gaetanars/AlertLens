# ADR-005 — Security Foundation (Phase 1)

**Status:** Accepted  
**Date:** 2026-03-10  
**Author:** Developer Agent  
**Implements:** GitHub issues #30 (YAML injection), #31 (auth bypass), #32 (CSRF), #33 (XSS)

---

## Context

AlertLens manages Alertmanager configurations that are deployed to production infrastructure.
A security compromise could silently disable alerting, redirect notifications, or expose sensitive
environment data.  Four attack vectors were identified in GitHub issues #30–#33:

| Issue | Vector | Severity |
|-------|---------|----------|
| #30 | YAML injection in config builder | Medium |
| #31 | Authentication bypass / privilege escalation | High |
| #32 | Cross-Site Request Forgery (CSRF) | High |
| #33 | Cross-Site Scripting (XSS) | High |

This ADR documents the architectural decisions made to address all four vectors.

---

## Decision 1 — YAML Validation via Official Alertmanager Library (#30)

### Problem
User-supplied YAML for Alertmanager configs could contain:
- Anchor bombs (billion-laughs DoS)
- Invalid field types that pass through silently
- Path traversal in `config_file_path`
- Oversized payloads causing OOM

### Decision
Use the **official `prometheus/alertmanager/config` parser** for all config validation
(`configbuilder.Validate`).  This is the same parser that Alertmanager itself uses, ensuring
100% compatibility and strict schema enforcement.

Additional hardening:
1. **Request body cap** — `http.MaxBytesReader` limits all incoming bodies to 10 MiB
   (`internal/api/router.go`).  Alertmanager configs are typically < 1 MiB.
2. **Strict YAML parser** — `gopkg.in/yaml.v3` is used everywhere; it rejects anchors
   that would expand beyond memory limits.
3. **Dedicated validate endpoint** — `POST /api/config/validate` runs the parser and
   returns `{ valid: bool, warnings: [], errors: [] }` without persisting anything.
4. **Role gate** — The config endpoints require `config-editor` role (see Decision 2).

### Alternatives Rejected
- Custom YAML schema validation: brittle, diverges from Alertmanager's own parser.
- Allowlist field names: too strict, would break legitimate advanced configs.

---

## Decision 2 — JWT Authentication with RBAC and TOTP MFA (#31)

### Problem
The original implementation used a single admin password (binary auth).  This is insufficient:
- No least-privilege: all users have full write access.
- No MFA for high-privilege operations.
- Tokens are never revoked on logout.

### Decision

#### 2a. Role-Based Access Control (RBAC)

Four roles form a strict privilege hierarchy:

```
viewer < silencer < config-editor < admin
```

| Role | Capabilities |
|------|-------------|
| `viewer` | Read alerts, silences, routing tree |
| `silencer` | viewer + create/update/expire silences |
| `config-editor` | silencer + read/write Alertmanager config |
| `admin` | config-editor + full control (future admin ops) |

Roles are expressed as a JWT `role` claim.  Each API route is protected by
`RequireRole(minimumRole)` middleware (`internal/auth/middleware.go`).

Route protection:
- `GET /api/*` (alerts, routing, silences list) — public / viewer-level
- `POST|PUT|DELETE /api/silences/*` — `RoleSilencer` or higher
- `GET|POST /api/config/*` — `RoleConfigEditor` or higher

#### 2b. JWT Design

- **Algorithm:** HS256 (HMAC-SHA256).  Symmetric key derived from the admin password SHA-256 hash,
  meaning tokens are automatically invalidated when the password changes.
- **Claims:** `sub` (role string), `role`, `iat`, `exp` (24 h TTL), `jti` (random 128-bit).
- **Revocation:** In-memory revocation set keyed by `jti`.  Entries are TTL-purged on each
  `Revoke()` call to prevent unbounded growth.  (Stateless design: restarts clear the set,
  which is acceptable given 24 h TTL and the target deployment model.)

#### 2c. TOTP Multi-Factor Authentication (MFA)

For users who have a `totp_secret` configured, the login flow requires a second factor:

```
POST /api/auth/login
Body: { "password": "...", "totp_code": "123456" }
```

If `totp_code` is absent and the user has MFA enabled, the response is:
```json
{ "error": "MFA challenge required", "mfa_required": true }
```
HTTP 401.

TOTP specification:
- **RFC 6238** (Time-Based OTP), 6 digits, SHA-1, 30-second window.
- **Clock skew tolerance:** ±1 step (±30 s) to handle minor drift.
- **Secret format:** Base32 (no padding), compatible with Google Authenticator, Authy, 1Password.
- **Storage:** Secret stored in `config.yaml` under `auth.users[].totp_secret`.
  Operators are responsible for keeping `config.yaml` confidential (file permissions, secrets manager).

Admin setup flow:
```
# Generate a TOTP secret (Go helper)
alertlens totp-setup --issuer AlertLens --account admin
# → Prints base32 secret + QR code URI
# Scan with authenticator app, add to config.yaml
```

#### 2d. Brute-Force Protection

Login endpoint is rate-limited per source IP: 5 attempts per minute with exponential
back-off (existing `ratelimit.go` unchanged).

### Alternatives Rejected
- OAuth2/OIDC: correct for enterprise, over-engineered for Phase 1.  Planned for Phase 2.
- Database-backed sessions: contradicts the stateless design principle.
- Hardware keys (FIDO2/WebAuthn): future enhancement (Phase 2+).

---

## Decision 3 — CSRF Protection via Signed Double-Submit Cookie (#32)

### Problem
Cross-site requests from attacker-controlled origins can trigger state-mutating API calls
(create silence, write config) using the victim's session cookie.

### Decision

**Pattern:** Signed double-submit cookie (OWASP recommended for SPAs).

**Implementation** (`internal/auth/csrf.go`):

1. On `GET/HEAD/OPTIONS`: generate a signed token, set it as a `SameSite=Lax` cookie
   (`csrf_token`) and echo it in the `X-CSRF-Token` response header.
2. On `POST/PUT/PATCH/DELETE`:
   - **Exempt** requests with a `Bearer` token: browsers cannot set `Authorization: Bearer …`
     cross-site without a preflight, making it an implicit CSRF defence.
   - **Others** (e.g. unauthenticated `POST /auth/login`): require the `X-CSRF-Token` header
     to match the `csrf_token` cookie.  Both are verified against the server HMAC secret to
     prevent cookie injection.

**Token format:** `<16-byte-random-hex>.<HMAC-SHA256(secret, random-hex)-hex>`

**TTL:** 8 hours (refreshed on every GET).

**Frontend integration** (`web/src/lib/api/client.ts`): reads the token from the
`X-CSRF-Token` response header on initial load and includes it in all mutating requests.

### Alternatives Rejected
- Synchronizer (stateful) tokens: requires server-side state, contradicts stateless design.
- `SameSite=Strict` cookies only: does not protect API-only endpoints served from different
  sub-paths; double-submit provides defence-in-depth.

---

## Decision 4 — XSS Prevention via Content Security Policy (#33)

### Problem
If an XSS vector is found in the SvelteKit frontend or reflected from the API, an attacker
could exfiltrate JWT tokens or trigger config mutations on behalf of the victim.

### Decision

**CSP Header** (`internal/api/router.go`):

```
Content-Security-Policy:
  default-src 'self';
  script-src 'self';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data: blob:;
  font-src 'self';
  connect-src 'self';
  object-src 'none';
  base-uri 'self';
  form-action 'self';
  frame-ancestors 'none';
```

- `script-src 'self'` — no inline scripts, no CDN, no `eval()`.
- `frame-ancestors 'none'` — prevents clickjacking (redundant with `X-Frame-Options: DENY`).
- `object-src 'none'` — eliminates Flash/PDF plugin vectors.

**Additional headers:**
- `X-Content-Type-Options: nosniff` — prevents MIME sniffing.
- `X-Frame-Options: DENY` — legacy framing protection.
- `Referrer-Policy: strict-origin-when-cross-origin` — limits referrer leakage.

**Frontend:**
- SvelteKit does not use `dangerouslySetInnerHTML` or `{@html}` with user data (audited ✓).
- All user-supplied label/annotation values are rendered via Svelte's default text-node binding,
  which escapes HTML automatically.

### Alternatives Rejected
- Nonce-based CSP: more secure but requires server-side rendering or build-time injection,
  incompatible with the current static SPA build.  Consider for Phase 2 if SSR is adopted.
- Report-only mode: useful for gradual rollout; not needed here as the policy is known-safe
  for the current frontend.

---

## Consequences

### Positive
- All four critical security vectors from Phase 1 are addressed.
- Zero-dependency CSRF (no third-party middleware).
- TOTP MFA is standards-compliant and works with all major authenticator apps.
- CSP blocks entire classes of XSS exploitation even if a reflection point is found.
- RBAC enables least-privilege deployments (read-only dashboards for NOC teams).

### Negative / Trade-offs
- Stateless JWT revocation is imperfect: server restart clears the revocation set.
  Acceptable for Phase 1; a Redis/database-backed revocation store is a Phase 2 option.
- `style-src 'unsafe-inline'` is required by Tailwind CSS (v4 injects styles at runtime).
  Switching to Tailwind CSS v4 with `@layer` + PostCSS can eliminate this in a future PR.
- TOTP secrets are stored in `config.yaml` in plaintext.  Operators must manage file
  permissions and consider a secrets manager integration (Phase 2).

---

## References

- [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [OWASP Content Security Policy](https://cheatsheetseries.owasp.org/cheatsheets/Content_Security_Policy_Cheat_Sheet.html)
- [RFC 6238 — TOTP](https://www.rfc-editor.org/rfc/rfc6238)
- [RFC 7519 — JWT](https://www.rfc-editor.org/rfc/rfc7519)
- [pquerna/otp](https://github.com/pquerna/otp) — TOTP library used
- [Prometheus Alertmanager config package](https://pkg.go.dev/github.com/prometheus/alertmanager/config)
