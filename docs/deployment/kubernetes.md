# Kubernetes (Helm)

AlertLens ships an official Helm chart published to the GitHub Container Registry (GHCR) as an OCI artifact.

## Prerequisites

- Kubernetes **1.21+**
- Helm **3.8+**
- A running Prometheus Alertmanager or Grafana Mimir instance reachable from within the cluster

---

## Installation

=== "Minimal (read-only)"

    Connect to a single Alertmanager without admin mode:

    ```bash
    helm install alertlens \
      oci://ghcr.io/gaetanars/charts/alertlens \
      --set alertlens.alertmanagers[0].url=http://alertmanager.monitoring:9093
    ```

=== "With admin password"

    Enables silence creation, visual acks, and configuration editing:

    ```bash
    helm install alertlens \
      oci://ghcr.io/gaetanars/charts/alertlens \
      --set alertlens.alertmanagers[0].url=http://alertmanager.monitoring:9093 \
      --set alertlens.adminPassword=your-strong-password
    ```

=== "From a values file"

    ```bash
    helm install alertlens \
      oci://ghcr.io/gaetanars/charts/alertlens \
      -f alertlens-values.yaml
    ```

Open in a browser with:

```bash
kubectl port-forward svc/alertlens 9000:9000
# → http://localhost:9000
```

---

## Minimal values.yaml

```yaml
alertlens:
  alertmanagers:
    - name: production
      url: http://alertmanager.monitoring:9093
  adminPassword: "your-strong-password"
```

---

## Common configurations

### Multiple Alertmanager instances

```yaml
alertlens:
  alertmanagers:
    - name: prod-eu
      url: http://alertmanager-eu.monitoring:9093
    - name: prod-us
      url: http://alertmanager-us.monitoring:9093
    - name: staging
      url: http://alertmanager-staging.monitoring:9093
      tls_skip_verify: true
```

### Grafana Mimir (multi-tenant)

```yaml
alertlens:
  alertmanagers:
    - name: platform
      url: http://mimir-alertmanager.monitoring:8080
      tenant_id: platform
    - name: apps
      url: http://mimir-alertmanager.monitoring:8080
      tenant_id: apps
```

### Ingress with TLS (nginx + cert-manager)

```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: alertlens.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: alertlens-tls
      hosts:
        - alertlens.example.com
```

### GitOps integration

```yaml
alertlens:
  gitops:
    github:
      token: "ghp_..."   # contents:write on the target repo
    # gitlab:
    #   token: "glpat-..."
    #   url: "https://gitlab.example.com"
```

---

## Sensitive values and secret management

By default, the chart creates a `Secret` containing the admin password and GitOps tokens.

For production environments or external secret management (External Secrets Operator, Sealed Secrets, Vault, etc.), create the secret yourself and reference it:

```yaml
# The secret must contain any combination of these keys:
#   admin-password   → ALERTLENS_AUTH_ADMIN_PASSWORD
#   github-token     → ALERTLENS_GITOPS_GITHUB_TOKEN
#   gitlab-token     → ALERTLENS_GITOPS_GITLAB_TOKEN
apiVersion: v1
kind: Secret
metadata:
  name: alertlens-credentials
type: Opaque
stringData:
  admin-password: "your-strong-password"
  github-token: "ghp_..."
```

Then reference it in values:

```yaml
existingSecret: alertlens-credentials
alertlens:
  alertmanagers:
    - name: production
      url: http://alertmanager:9093
```

### Per-instance credentials

Alertmanager basic_auth passwords can be injected via `extraEnv` to avoid storing them in the ConfigMap:

```yaml
extraEnv:
  - name: ALERTLENS_ALERTMANAGERS_0_BASIC_AUTH_PASSWORD
    valueFrom:
      secretKeyRef:
        name: alertmanager-credentials
        key: password
```

---

## High availability

AlertLens is fully stateless — scale it freely.

```yaml
replicaCount: 2

podDisruptionBudget:
  enabled: true
  minAvailable: 1

topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: alertlens
```

---

## Upgrading

```bash
helm upgrade alertlens oci://ghcr.io/gaetanars/charts/alertlens \
  --reuse-values \
  --version 0.2.0
```

---

## Uninstalling

```bash
helm uninstall alertlens
```

!!! warning "Secret is deleted on uninstall"
    The chart-managed `Secret` is deleted with the release. Use `existingSecret` if you need to retain credentials across uninstalls.

---

## Security posture

The chart enforces Kubernetes security best practices out of the box:

| Control | Value |
|---|---|
| Run as non-root | `runAsNonRoot: true`, UID `65534` |
| Read-only root filesystem | `true` (writable `/tmp` via emptyDir) |
| Privilege escalation | `allowPrivilegeEscalation: false` |
| Linux capabilities | All dropped |
| Seccomp profile | `RuntimeDefault` |
| ServiceAccount token | Not auto-mounted |

---

## Full values reference

See the [chart README](https://github.com/gaetanars/AlertLens/blob/main/charts/alertlens/README.md) or run:

```bash
helm show values oci://ghcr.io/gaetanars/charts/alertlens
```
