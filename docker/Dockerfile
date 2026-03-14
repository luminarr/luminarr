# ── Frontend build stage ──────────────────────────────────────────────────────
FROM node:22-alpine AS frontend

WORKDIR /app/web/ui

# Install dependencies first (cached unless package files change)
COPY web/ui/package.json web/ui/package-lock.json ./
RUN npm ci

# Copy frontend source and build
COPY web/ui/ ./
RUN npm run build
# Output lands in /app/web/static/

# ── Go build stage ───────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG TMDB_API_KEY

WORKDIR /app

# Download dependencies first (cached unless go.mod/go.sum change)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Overwrite web/static/ with the freshly built frontend
COPY --from=frontend /app/web/static/ ./web/static/

# modernc/sqlite is pure Go — CGO_ENABLED=0 produces a fully static binary.
RUN CGO_ENABLED=0 GOOS=linux go build \
      -ldflags "-X github.com/luminarr/luminarr/internal/version.Version=${VERSION} \
                -X github.com/luminarr/luminarr/internal/version.BuildTime=${BUILD_TIME} \
                -X github.com/luminarr/luminarr/internal/config.DefaultTMDBKey=${TMDB_API_KEY} \
                -s -w" \
      -o /luminarr ./cmd/luminarr

# ── Final stage ───────────────────────────────────────────────────────────────
# Alpine with ffmpeg (includes ffprobe) for media scanning out of the box.
FROM alpine:3.21

# ffmpeg pulls in ffprobe. ca-certificates is needed for outbound HTTPS.
# tzdata provides timezone data for correct time display.
RUN apk add --no-cache ffmpeg ca-certificates tzdata && \
    adduser -D -u 1000 luminarr

# The binary
COPY --from=builder /luminarr /luminarr

# Default data directory (mount a volume here in production)
VOLUME ["/config"]

EXPOSE 8282

USER luminarr

ENTRYPOINT ["/luminarr"]
