# Getting Started

## Prerequisites

- A running **Prometheus Alertmanager** (or Grafana Mimir) instance accessible from the machine running AlertLens.
- **Alertmanager API v2** must be enabled (default since Alertmanager 0.21).

---

## Installation

=== "Docker (recommended)"

    Pull and run the official image:

    ```bash
    docker run -d \
      --name alertlens \
      -p 9000:9000 \
      -v $(pwd)/alertlens.yaml:/etc/alertlens/alertlens.yaml:ro \
      ghcr.io/gaetanars/alertlens:latest \
      -config /etc/alertlens/alertlens.yaml
    ```

    See [Docker deployment](deployment/docker.md) for a full Docker Compose example.

=== "Pre-built binary"

    Download the latest binary from the [GitHub Releases](https://github.com/gaetanars/AlertLens/releases) page:

    ```bash
    # Linux amd64
    curl -LO https://github.com/gaetanars/AlertLens/releases/latest/download/alertlens-linux-amd64
    chmod +x alertlens-linux-amd64
    ./alertlens-linux-amd64 -config alertlens.yaml
    ```

=== "Build from source"

    Requires Go 1.24+ and Node.js 20+.

    ```bash
    git clone https://github.com/gaetanars/AlertLens.git
    cd AlertLens
    make build
    ./alertlens -config alertlens.yaml
    ```

    See [Binary / From Source](deployment/binary.md) for details.

---

## Minimal Configuration

Create an `alertlens.yaml` file:

```yaml
alertmanagers:
  - name: production
    url: http://alertmanager.example.com:9093
```

That's it. Start AlertLens and open [http://localhost:9000](http://localhost:9000).

!!! info "No admin password"
    Without `auth.admin_password`, AlertLens runs in **read-only mode**: you can view alerts, silences, and the routing tree, but cannot create silences or edit configurations.

---

## Enable Admin Mode

To unlock silence creation, visual acks, and configuration editing, set an admin password:

```yaml
auth:
  admin_password: "your-strong-password"

alertmanagers:
  - name: production
    url: http://alertmanager.example.com:9093
```

Or via environment variable:

```bash
export ALERTLENS_AUTH_ADMIN_PASSWORD="your-strong-password"
```

Then log in from the AlertLens UI.

---

## Multiple Alertmanager Instances

AlertLens can aggregate alerts from several instances simultaneously:

```yaml
alertmanagers:
  - name: prod-eu
    url: http://alertmanager-eu:9093

  - name: prod-us
    url: http://alertmanager-us:9093

  - name: staging
    url: http://alertmanager-staging:9093
    tls_skip_verify: true
```

The UI provides a source indicator per alert and an instance filter.

---

## Configuration Precedence

```
Default values  →  Config file  →  Environment variables
                                   (highest priority)
```

Environment variables follow the pattern `ALERTLENS_<SECTION>_<KEY>`.
Examples:

| Config key | Environment variable |
|---|---|
| `server.port` | `ALERTLENS_SERVER_PORT` |
| `auth.admin_password` | `ALERTLENS_AUTH_ADMIN_PASSWORD` |
| `gitops.github.token` | `ALERTLENS_GITOPS_GITHUB_TOKEN` |

See the full [Configuration reference](configuration.md).

---

## Next Steps

- [Configuration reference](configuration.md) — all options explained
- [Alert Visualization](features/visualization.md) — filters, grouping, multi-instance
- [Docker deployment](deployment/docker.md) — production-ready setup
