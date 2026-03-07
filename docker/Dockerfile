# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG TMDB_API_KEY

WORKDIR /app

# Download dependencies first (cached unless go.mod/go.sum change)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# modernc/sqlite is pure Go — CGO_ENABLED=0 produces a fully static binary.
RUN CGO_ENABLED=0 GOOS=linux go build \
      -ldflags "-X github.com/luminarr/luminarr/internal/version.Version=${VERSION} \
                -X github.com/luminarr/luminarr/internal/version.BuildTime=${BUILD_TIME} \
                -X github.com/luminarr/luminarr/internal/config.DefaultTMDBKey=${TMDB_API_KEY} \
                -s -w" \
      -o /luminarr ./cmd/luminarr

# Create a minimal /etc/passwd so the scratch image has a non-root user.
# UID/GID 65532 matches the distroless "nonroot" convention.
RUN echo "nonroot:x:65532:65532:nonroot:/:/sbin/nologin" > /etc/passwd.luminarr && \
    echo "nonroot:x:65532:" > /etc/group.luminarr

# ── Final stage ───────────────────────────────────────────────────────────────
# scratch: zero OS footprint. The binary is fully self-contained.
FROM scratch

# TLS root certificates (needed for outbound HTTPS: TMDB, indexers, Claude API)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Minimal passwd/group so the process can run as a named non-root user.
COPY --from=builder /etc/passwd.luminarr /etc/passwd
COPY --from=builder /etc/group.luminarr /etc/group

# The binary
COPY --from=builder /luminarr /luminarr

# Default data directory (mount a volume here in production)
VOLUME ["/config"]

EXPOSE 8282

# Run as non-root. The /config volume must be owned by UID 65532 on the host,
# or mounted with appropriate permissions, for the process to write to it.
# docker-compose.yml sets LUMINARR_DATABASE_PATH and /config is the volume.
USER nonroot:nonroot

ENTRYPOINT ["/luminarr"]
