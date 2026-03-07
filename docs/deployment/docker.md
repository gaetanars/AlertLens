# Docker Deployment

The recommended way to run AlertLens in production is via Docker.

---

## Quick Run

```bash
docker run -d \
  --name alertlens \
  -p 9000:9000 \
  -e ALERTLENS_AUTH_ADMIN_PASSWORD="your-strong-password" \
  -e ALERTLENS_ALERTMANAGERS_0_URL="http://alertmanager:9093" \
  --restart unless-stopped \
  ghcr.io/alertlens/alertlens:latest
```

---

## Using a Config File

Create an `alertlens.yaml` file:

```yaml
server:
  port: 9000

auth:
  admin_password: "your-strong-password"

alertmanagers:
  - name: production
    url: http://alertmanager.internal:9093
```

Then mount it into the container:

```bash
docker run -d \
  --name alertlens \
  -p 9000:9000 \
  -v $(pwd)/alertlens.yaml:/etc/alertlens/alertlens.yaml:ro \
  ghcr.io/alertlens/alertlens:latest \
  -config /etc/alertlens/alertlens.yaml
```

---

## Docker Compose

Production-ready example with Alertmanager and AlertLens:

```yaml
services:
  alertmanager:
    image: prom/alertmanager:v0.27.0
    command:
      - --config.file=/etc/alertmanager/alertmanager.yml
      - --storage.path=/alertmanager
    volumes:
      - ./alertmanager:/etc/alertmanager
      - alertmanager-data:/alertmanager
    ports:
      - "9093:9093"
    restart: unless-stopped

  alertlens:
    image: ghcr.io/alertlens/alertlens:latest
    command: [-config, /etc/alertlens/alertlens.yaml]
    volumes:
      - ./alertlens.yaml:/etc/alertlens/alertlens.yaml:ro
      # Mount alertmanager config dir if using disk-write mode:
      - ./alertmanager:/etc/alertmanager
    ports:
      - "9000:9000"
    depends_on:
      - alertmanager
    restart: unless-stopped
    environment:
      ALERTLENS_AUTH_ADMIN_PASSWORD: "${ALERTLENS_ADMIN_PASSWORD}"
      ALERTLENS_GITOPS_GITHUB_TOKEN: "${GITHUB_TOKEN}"

volumes:
  alertmanager-data:
```

---

## Available Tags

| Tag | Description |
|---|---|
| `latest` | Latest stable release |
| `vX.Y.Z` | Specific version |
| `dev` | Built from the `main` branch (unstable) |

---

## Configuration Builder + Disk Write

To use the Configuration Builder's disk-write mode, AlertLens needs write access to the `alertmanager.yml` file. Share the same directory mount between both containers:

```yaml
services:
  alertmanager:
    volumes:
      - ./alertmanager:/etc/alertmanager   # writable

  alertlens:
    volumes:
      - ./alertmanager:/etc/alertmanager   # same mount
```

And set `config_file_path` in your AlertLens config:

```yaml
alertmanagers:
  - name: production
    url: http://alertmanager:9093
    config_file_path: /etc/alertmanager/alertmanager.yml
```

---

## Networking

AlertLens only makes **outbound** requests to Alertmanager and to Git providers (GitHub / GitLab). It does not require incoming connectivity beyond the UI/API port (default `9000`).

In Docker Compose, make sure AlertLens and Alertmanager are on the same network (Docker Compose does this automatically within a `docker-compose.yml`).

---

## Health Check

AlertLens exposes a health endpoint:

```
GET /api/v1/health
```

Use it as a Docker health check:

```yaml
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://localhost:9000/api/v1/health"]
  interval: 30s
  timeout: 5s
  retries: 3
```

!!! note "Distroless base image"
    The official image is built on `gcr.io/distroless/static-debian12`, which does not include a shell. Use an `exec`-form health check as shown above.
