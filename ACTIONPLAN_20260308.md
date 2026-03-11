# 🚀 AlertLens Phase 1 - Plan d'Action Immédiat
**Date:** 2026-03-08 | **Gaëtan:** Progression visible dès demain

---

## 📊 État Actuel - Analyse Code

### Backend (Go) ✅ 70% complet
- **Config loading:** ✅ Safe (uses `gopkg.in/yaml.v3`)
- **Authentication:** ✅ JWT implemented, rate-limited login
- **API routes:** ✅ Structured with chi router, admin middleware applied
- **Alertmanager pool:** ✅ Multi-instance support
- **Config validation:** ✅ Uses official `prometheus/alertmanager` config parser
- **Storage modes:** ✅ Disk + GitHub + GitLab ready

### Frontend (SvelteKit) ✅ 50% complet
- Alert visualization components exist (Kanban, Table)
- Filters, bulk actions scaffolded
- Auth store implemented
- Config builder partially done

### Missing/TODO
- ❌ RBAC system (users with different roles)
- ❌ Audit logging for config changes (#23)
- ❌ CSRF protection middleware (#32)
- ❌ XSS hardening in frontend (#33)
- ❌ Config history/rollback (#23)
- ❌ Notification template management (#22)
- ❌ Context-aware alert details (#20)

---

## 🔒 Security Gaps Identified

| Issue | Severity | Status | Quick Win |
|-------|----------|--------|-----------|
| #30 YAML Injection | Medium | SAFE NOW* | Add formal validation rules in PR |
| #31 Auth Bypass | High | Partial** | Implement RBAC + session hardening |
| #32 CSRF | High | Missing | Add sync-token middleware (5h) |
| #33 XSS | High | Missing | CSP header + output encoding (8h) |

**Notes:**
- *YAML parsing is safe (official Alertmanager lib), but needs formal input sanitization spec
- **Current code: admin/non-admin binary. Needs granular roles (viewer, editor, admin)

---

## 📋 Phase 1 Issues - Prioritized

### TIER 1 (Foundation - Blocker for others)
1. **#24 RBAC** - Prerequisite for audit, config save permissions
   - Current: Binary admin auth only
   - Needed: Role system (viewer, editor, admin)
   - Effort: 2 days (backend) + 1 day (frontend)

2. **#32 CSRF Protection** - Security must-have
   - Add `csrf` middleware to chi router
   - SameSite=Strict on auth tokens
   - Effort: 1 day

3. **#33 XSS Prevention** - Security must-have
   - Add CSP header in router
   - Audit frontend for `dangerouslySetInnerHTML` usage
   - Effort: 1.5 days

### TIER 2 (Core features - visible progress)
4. **#21 Config Editor** - Visual UX for routing/receivers
   - Partially done, needs frontend polish
   - Effort: 3 days

5. **#23 Config History** - Audit trail + rollback
   - Requires git log parsing (if using git mode) or simple file backup
   - Effort: 2 days

6. **#20 Context Alerts** - Show logs/metrics for alert
   - Requires Prometheus/Loki integration research
   - Effort: 3 days (research + MVP)

### TIER 3 (Nice-to-have)
7. **#22 Notification Templates** - Template YAML editor
   - Effort: 2 days

---

## 🎯 Immediate Action Plan (Next 48h)

### For **ARCHITECT** (2h, start now)
1. Design RBAC model:
   - Roles: `viewer`, `silencer`, `config-editor`, `admin`
   - Permissions matrix
   - Session/token strategy
   - **Deliverable:** `ADR-RBAC.md` (GitHub issue comment)

2. Sketch security hardening checklist:
   - CSRF token injection strategy
   - CSP policy for SvelteKit
   - JWT claims validation
   - **Deliverable:** `SECURITY-HARDENING.md`

### For **SECURITY** (if available, 1h)
- Review ADR-RBAC for gaps
- Approve YAML validation spec
- Sign off on CSRF/XSS strategy

### For **DEVELOPER** (Starting tonight, Priority 1)
**CONCRETE TASK:** Implement RBAC + CSRF Protection

#### Step 1: RBAC Backend (4h)
**File:** `internal/auth/rbac.go` (new)

Create:
```go
type Role string
const (
    RoleViewer      Role = "viewer"       // Read-only (alerts, silences, routing)
    RoleSilencer    Role = "silencer"     // Create/edit silences + acks
    RoleConfigEditor Role = "config_editor" // Edit AM config
    RoleAdmin       Role = "admin"        // Full access
)

// Add to auth.Service:
- ValidateWithRole(token string) (jti string, role Role, err error)
- SetUserRole(jti string, role Role)  // Store in jwt claims
```

**Update:** `internal/api/router.go`
- Apply role-based middleware to endpoints:
  - `/silences/*` → require `RoleSilencer|RoleConfigEditor|RoleAdmin`
  - `/config/*` → require `RoleConfigEditor|RoleAdmin`
  - `/routing/*` → require `RoleViewer+`
  - `/health, /auth/status, /auth/login, /alerts` → public

**Effort:** 4 hours | **Test:** Unit tests for role validation

#### Step 2: CSRF Protection (2h)
**File:** `internal/api/middleware/csrf.go` (new)

Add to router:
```go
r.Use(csrf.Middleware(...))  // Generate token on GET /api/csrf
// Validate on POST/PUT/DELETE to /api/config/*, /api/silences/*
```

Modify handler responses:
- Include `X-CSRF-Token` in all API responses
- Frontend reads & sends in `X-CSRF-Token` header

**Effort:** 2 hours | **Test:** CSRF token validation tests

#### Step 3: Basic XSS Hardening (1h)
**File:** `internal/api/router.go`

Add CSP header:
```go
w.Header().Set("Content-Security-Policy", 
  "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'")
```

No `dangerouslySetInnerHTML` in current code (checked ✓)

**Effort:** 1 hour

#### Step 4: Frontend Auth Integration (2h)
**File:** `web/src/lib/api/auth.ts`

Add:
- Get roles from JWT claims
- Store `role` in auth store
- Pass role to components for conditional rendering

Update config handlers to validate role before calling API

**Effort:** 2 hours

---

## 🏁 Definition of Done - Visible Progress for Gaëtan

After 9 hours (1 developer day):
- ✅ RBAC system operational (4 roles, middleware protecting endpoints)
- ✅ CSRF tokens generated & validated
- ✅ CSP header in all responses
- ✅ Frontend respects roles (UI hides config buttons for non-editors)
- ✅ All changes in single PR with tests
- ✅ Demo: Login as "viewer" → no edit buttons; "config-editor" → config panel visible

**GitHub:**
- New draft PR: "feat: RBAC + security hardening for Phase 1"
- Closes: #24, #32, #33 (partial)
- Visible progress: Role system + protected endpoints

---

## 📦 Follow-up (Day 2-3)

### Developer continues:
- Config history storage (#23) - 2h
- Notification template editor UI (#22) - 4h
- Alert context (Prometheus integration) (#20) - 4h

### Architect:
- Frontend architecture for config builder refinement
- Data fetching strategy for context (Prometheus/Loki)

---

## 🎓 Lessons from Code Review

### Strengths ✨
- Solid auth implementation (JWT + rate limit on login)
- Safe YAML parsing (official libs)
- Clean Go structure, middleware pattern working well
- Multi-instance support designed correctly
- Git integration scaffolded

### Weaknesses ⚠️
- No granular permissions (binary admin/non-admin)
- No request-level security headers (CSRF, CSP)
- Frontend components not role-aware
- No config change audit trail
- No input validation spec for JSON payloads (currently relies on struct tags)

### Next Security Pass
After RBAC:
1. Input validation middleware (JSON schemas)
2. Rate limiting per role
3. Audit logging (who, what, when, config diff)
4. Session revocation strategy for config-heavy ops

---

## 🔗 Links & Files

- **Repo:** AlertLens/AlertLens
- **Code:** `/tmp/AlertLens/` (analyzed ✓)
- **Issues:** #20-24 (Phase 1), #30-33 (Security)
- **Architecture:** SPECS.md reviewed
- **Current auth:** `internal/auth/auth.go` (JWT, rate limit) ✓
- **Router:** `internal/api/router.go` (middleware ready) ✓

---

## 🚦 Next Steps

1. **Architect:** Comment on #24 with RBAC ADR (2h)
2. **Developer:** Clone repo, create feature branch `feat/rbac-csrf`, start coding (tonight, 9h commitment)
3. **Gaëtan:** Expect draft PR tomorrow with RBAC + CSRF framework ready for review

**Goal:** By EOD tomorrow, AlertLens Phase 1 has security foundation. Config editor can be tested with proper access control.

---

Generated by: Planner Agent | 2026-03-08
