# AlertLens Helm Chart

A modern UI for Prometheus Alertmanager — visualize, silence, and manage configurations with ease.

## TL;DR

```bash
helm install alertlens oci://ghcr.io/gaetanars/chart/alertlens \
  --set alertlens.alertmanagers[0].url=http://alertmanager:9093
```

## Introduction

This chart deploys [AlertLens](https://github.com/gaetanars/AlertLens) on a Kubernetes cluster.

AlertLens is a **stateless** single binary — all state lives in Alertmanager. The chart deploys:

- A `Deployment` running the AlertLens container
- A `ConfigMap` holding the non-sensitive configuration (`alertlens.yaml`)
- A `Secret` holding the admin password and GitOps tokens
- A `Service` (ClusterIP by default)
- Optionally: `Ingress`, `HorizontalPodAutoscaler`, `PodDisruptionBudget`

## Prerequisites

- Kubernetes 1.21+
- Helm 3.8+
- A running Prometheus Alertmanager or Grafana Mimir instance reachable from the cluster

## Installing the chart

```bash
# Minimal — read-only mode, no admin password
helm install alertlens oci://ghcr.io/gaetanars/chart/alertlens \
  --set alertlens.alertmanagers[0].url=http://alertmanager.monitoring:9093

# With admin password (enables silence creation and config editing)
helm install alertlens oci://ghcr.io/gaetanars/chart/alertlens \
  --set alertlens.alertmanagers[0].url=http://alertmanager.monitoring:9093 \
  --set alertlens.adminPassword=my-strong-password
```

Or with a `values.yaml` file:

```bash
helm install alertlens oci://ghcr.io/gaetanars/chart/alertlens -f values.yaml
```

## Uninstalling the chart

```bash
helm uninstall alertlens
```

## Configuration

### Minimal values.yaml

```yaml
alertlens:
  alertmanagers:
    - name: production
      url: http://alertmanager.monitoring:9093
  adminPassword: "your-strong-password"
```

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
    - name: mimir-platform
      url: http://mimir-alertmanager.monitoring:8080
      tenant_id: platform
    - name: mimir-apps
      url: http://mimir-alertmanager.monitoring:8080
      tenant_id: apps
```

### Ingress with TLS

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

### Using an existing Secret

For GitOps or external secret management:

```yaml
existingSecret: my-alertlens-secret
alertlens:
  alertmanagers:
    - name: production
      url: http://alertmanager:9093
```

The secret must contain any combination of:

| Key | Environment variable |
|---|---|
| `admin-password` | `ALERTLENS_AUTH_ADMIN_PASSWORD` |
| `github-token` | `ALERTLENS_GITOPS_GITHUB_TOKEN` |
| `gitlab-token` | `ALERTLENS_GITOPS_GITLAB_TOKEN` |

### GitOps integration

```yaml
alertlens:
  gitops:
    github:
      token: "ghp_..."   # or use existingSecret
    # gitlab:
    #   token: "glpat-..."
    #   url: "https://gitlab.example.com"
```

### High availability

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

### Autoscaling

AlertLens is stateless and scales horizontally without configuration.

```yaml
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 80
```

## Parameters

### Image

| Parameter | Description | Default |
|---|---|---|
| `image.repository` | Image repository | `ghcr.io/gaetanars/alertlens` |
| `image.tag` | Image tag (defaults to chart AppVersion) | `""` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Image pull secrets | `[]` |

### General

| Parameter | Description | Default |
|---|---|---|
| `replicaCount` | Number of replicas | `1` |
| `nameOverride` | Override the chart name | `""` |
| `fullnameOverride` | Override the full name | `""` |

### AlertLens application

| Parameter | Description | Default |
|---|---|---|
| `alertlens.adminPassword` | Admin password (empty = read-only mode) | `""` |
| `alertlens.alertmanagers` | List of Alertmanager/Mimir instances | See values.yaml |
| `alertlens.gitops.github.token` | GitHub personal access token | `""` |
| `alertlens.gitops.gitlab.token` | GitLab personal access token | `""` |
| `alertlens.gitops.gitlab.url` | GitLab base URL | `https://gitlab.com` |
| `existingSecret` | Name of an existing Secret with sensitive keys | `""` |

### Service

| Parameter | Description | Default |
|---|---|---|
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `9000` |
| `service.annotations` | Service annotations | `{}` |

### Ingress

| Parameter | Description | Default |
|---|---|---|
| `ingress.enabled` | Enable Ingress | `false` |
| `ingress.className` | IngressClass name | `""` |
| `ingress.annotations` | Ingress annotations | `{}` |
| `ingress.hosts` | Ingress host rules | See values.yaml |
| `ingress.tls` | Ingress TLS configuration | `[]` |

### Resources and scaling

| Parameter | Description | Default |
|---|---|---|
| `resources.limits.cpu` | CPU limit | `200m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `50m` |
| `resources.requests.memory` | Memory request | `64Mi` |
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | HPA minimum replicas | `1` |
| `autoscaling.maxReplicas` | HPA maximum replicas | `3` |
| `podDisruptionBudget.enabled` | Enable PDB | `false` |

### Security

| Parameter | Description | Default |
|---|---|---|
| `podSecurityContext.runAsNonRoot` | Run as non-root | `true` |
| `podSecurityContext.runAsUser` | UID to run as | `65534` |
| `securityContext.readOnlyRootFilesystem` | Read-only root filesystem | `true` |
| `securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |

### Extras

| Parameter | Description | Default |
|---|---|---|
| `extraEnv` | Extra environment variables | `[]` |
| `extraEnvFrom` | Extra envFrom sources | `[]` |
| `extraVolumes` | Extra volumes | `[]` |
| `extraVolumeMounts` | Extra volume mounts | `[]` |

## Security

The chart follows Kubernetes security best practices out of the box:

- Runs as UID 65534 (nobody), non-root
- Read-only root filesystem (`/tmp` is an emptyDir)
- All Linux capabilities dropped
- `allowPrivilegeEscalation: false`
- RuntimeDefault seccomp profile
- ServiceAccount API credentials not auto-mounted

## Source code

- [AlertLens](https://github.com/gaetanars/AlertLens)
- [Helm chart](https://github.com/gaetanars/AlertLens/tree/main/chart/alertlens)

## License

[Apache 2.0](https://github.com/gaetanars/AlertLens/blob/main/LICENSE)
