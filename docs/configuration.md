# Configuration Reference

AlertLens is configured via a YAML file. Every key has a default value and can be overridden by an environment variable.

## Precedence

```
Default values  →  Config file  →  Environment variables
```

Environment variables follow the pattern `ALERTLENS_<SECTION>_<KEY>` (uppercase, underscores).

---

## Full Example

```yaml
server:
  host: "0.0.0.0"          # ALERTLENS_SERVER_HOST
  port: 9000                # ALERTLENS_SERVER_PORT
  cors_allowed_origins: []  # ALERTLENS_SERVER_CORS_ALLOWED_ORIGINS

auth:
  admin_password: ""        # ALERTLENS_AUTH_ADMIN_PASSWORD
  # If empty, admin mode is disabled (read-only access only)

alertmanagers:
  - name: "default"
    url: "http://localhost:9093"
    basic_auth:
      username: ""
      password: ""
    # Grafana Mimir multi-tenant: sent as X-Scope-OrgID header
    tenant_id: ""
    tls_skip_verify: false
    # Module C: path to alertmanager.yml on disk (disk-write mode)
    config_file_path: ""

gitops:
  github:
    token: ""               # ALERTLENS_GITOPS_GITHUB_TOKEN
  gitlab:
    token: ""               # ALERTLENS_GITOPS_GITLAB_TOKEN
    url: "https://gitlab.com"
```

---

## `server`

| Key | Type | Default | Description |
|---|---|---|---|
| `host` | string | `0.0.0.0` | Address to bind to. Use `127.0.0.1` to expose only locally. |
| `port` | int | `9000` | TCP port to listen on. |
| `cors_allowed_origins` | list | `[]` | List of allowed CORS origins. Only needed when the frontend is served from a different origin than the API. |

---

## `auth`

| Key | Type | Default | Description |
|---|---|---|---|
| `admin_password` | string | `""` | Password for the admin account. If empty, admin mode is completely disabled and the UI is read-only. |

!!! warning "Secure your password"
    Set this via the `ALERTLENS_AUTH_ADMIN_PASSWORD` environment variable rather than storing it in a file committed to version control.

---

## `alertmanagers`

A list of Alertmanager (or Mimir) instances. At least one is required.

| Key | Type | Default | Description |
|---|---|---|---|
| `name` | string | — | Display name for this instance. Must be unique. |
| `url` | string | — | Base URL of the Alertmanager API (without `/api/v2`). |
| `basic_auth.username` | string | `""` | HTTP Basic Auth username. |
| `basic_auth.password` | string | `""` | HTTP Basic Auth password. |
| `tenant_id` | string | `""` | Grafana Mimir tenant ID. Sent as the `X-Scope-OrgID` header on every request. |
| `tls_skip_verify` | bool | `false` | Disable TLS certificate verification. **Use only in trusted internal environments.** |
| `config_file_path` | string | `""` | Absolute path to the `alertmanager.yml` file on disk. Required for disk-write mode in the Configuration Builder. |

### Multi-instance example

```yaml
alertmanagers:
  - name: prod-eu
    url: http://alertmanager-eu.internal:9093

  - name: prod-us
    url: http://alertmanager-us.internal:9093
    basic_auth:
      username: alertlens
      password: secret

  - name: mimir-staging
    url: http://mimir.internal:9009
    tenant_id: platform-team
    tls_skip_verify: true
```

---

## `gitops`

Credentials for pushing Alertmanager configuration changes to a Git provider. Both GitHub and GitLab are supported simultaneously.

### `gitops.github`

| Key | Type | Default | Description |
|---|---|---|---|
| `token` | string | `""` | GitHub Personal Access Token (or fine-grained token) with `repo` scope. |

### `gitops.gitlab`

| Key | Type | Default | Description |
|---|---|---|---|
| `token` | string | `""` | GitLab Personal Access Token with `api` scope. |
| `url` | string | `https://gitlab.com` | GitLab instance URL (for self-hosted GitLab). |

!!! note "GitOps vs disk-write"
    The save strategy (GitOps push vs disk write) is configured per-instance in the Configuration Builder UI, not in this file. These tokens simply make the GitOps options available.
