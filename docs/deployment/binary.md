# Binary / From Source

## Download a Pre-built Binary

Pre-built binaries for Linux, macOS, and Windows are available on the [GitHub Releases](https://github.com/gaetanars/AlertLens/releases) page.

=== "Linux (amd64)"

    ```bash
    curl -LO https://github.com/gaetanars/AlertLens/releases/latest/download/alertlens-linux-amd64
    chmod +x alertlens-linux-amd64
    sudo mv alertlens-linux-amd64 /usr/local/bin/alertlens
    ```

=== "Linux (arm64)"

    ```bash
    curl -LO https://github.com/gaetanars/AlertLens/releases/latest/download/alertlens-linux-arm64
    chmod +x alertlens-linux-arm64
    sudo mv alertlens-linux-arm64 /usr/local/bin/alertlens
    ```

=== "macOS (arm64)"

    ```bash
    curl -LO https://github.com/gaetanars/AlertLens/releases/latest/download/alertlens-darwin-arm64
    chmod +x alertlens-darwin-arm64
    sudo mv alertlens-darwin-arm64 /usr/local/bin/alertlens
    ```

---

## Build from Source

### Prerequisites

| Tool | Minimum version |
|---|---|
| Go | 1.25 |
| Node.js | 20 |
| npm | 9 |

### Steps

```bash
# 1. Clone the repository
git clone https://github.com/gaetanars/AlertLens.git
cd AlertLens

# 2. Build frontend + backend (one command)
make build

# 3. Run
./alertlens -config alertlens.yaml
```

The `make build` target:

1. Runs `npm ci && npm run build` in the `web/` directory → produces `dist/`
2. Compiles Go with `go:embed all:dist` → bundles the frontend into the binary

The result is a single self-contained binary with no external dependencies.

### Available Make Targets

| Target | Description |
|---|---|
| `make build` | Full build (frontend + backend) |
| `make web-build` | Frontend only |
| `make go-build` | Go binary only (requires `dist/` to exist) |
| `make dev-backend` | Run backend with `config.example.yaml` |
| `make dev-frontend` | Run Vite dev server for frontend hot-reload |
| `make test` | Run Go tests with coverage |
| `make clean` | Remove build artifacts |

### Development Mode

For frontend development with hot-reload, run backend and frontend in separate terminals:

```bash
# Terminal 1 – Go backend
make dev-backend

# Terminal 2 – SvelteKit dev server (proxy to :9000)
make dev-frontend
```

---

## Running as a systemd Service

```ini
[Unit]
Description=AlertLens — Alertmanager UI
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=alertlens
ExecStart=/usr/local/bin/alertlens -config /etc/alertlens/alertlens.yaml
Restart=on-failure
RestartSec=5s
# Prevent config secrets from appearing in process list
EnvironmentFile=-/etc/alertlens/alertlens.env

[Install]
WantedBy=multi-user.target
```

Put secrets in `/etc/alertlens/alertlens.env`:

```env
ALERTLENS_AUTH_ADMIN_PASSWORD=your-strong-password
ALERTLENS_GITOPS_GITHUB_TOKEN=ghp_...
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now alertlens
```

---

## CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-config` | `""` | Path to the YAML config file. If omitted, defaults and environment variables are used. |

All configuration options are also available as environment variables — see the [Configuration reference](../configuration.md).
