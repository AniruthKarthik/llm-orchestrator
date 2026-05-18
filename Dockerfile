# =============================================================================
# Stage 1: Build the Go backend binary
# =============================================================================
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

# Install ca-certificates for HTTPS provider calls
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Statically-linked binary, no CGO
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags '-static'" \
    -o /app/server ./cmd/server

# =============================================================================
# Stage 2: Build the React UI
# =============================================================================
FROM node:22-alpine AS ui-builder

WORKDIR /ui

COPY ui/package*.json ./
RUN npm ci --prefer-offline

COPY ui/ .

# VITE_API_URL should be set at build time or left empty (uses /api/v1 relative path)
ARG VITE_API_URL=""
ENV VITE_API_URL=${VITE_API_URL}

RUN npm run build

# =============================================================================
# Stage 3: Minimal production image
# =============================================================================
FROM alpine:3.20

WORKDIR /app

# Security: run as non-root
RUN addgroup -S app && adduser -S app -G app

RUN apk add --no-cache ca-certificates tzdata

# Backend binary
COPY --from=go-builder /app/server /app/server
# Database migrations
COPY --from=go-builder /app/migrations /app/migrations
# Built UI (served by the backend's static file handler)
COPY --from=ui-builder /ui/dist /app/ui/dist

RUN chown -R app:app /app
USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

CMD ["/app/server"]
