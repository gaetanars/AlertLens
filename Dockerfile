# ─── Stage 1: Build frontend ────────────────────────────────────────────────
FROM node:25-alpine AS web-builder
WORKDIR /web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ─── Stage 2: Build Go binary ───────────────────────────────────────────────
FROM golang:1.25-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# svelte.config.js outputs to ../dist (relative to web/), i.e. /dist in the builder stage
COPY --from=web-builder /dist ./dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o alertlens .

# ─── Stage 3: Final image ───────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian13
COPY --from=go-builder /app/alertlens /alertlens
EXPOSE 9000
ENTRYPOINT ["/alertlens"]
