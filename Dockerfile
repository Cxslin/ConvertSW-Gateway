# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26
ARG NODE_VERSION=20

# Stage 1: build the WebUI assets.
FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-bookworm-slim AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: build the Go backend.
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS backend-builder
WORKDIR /src
COPY backend/go.mod backend/go.sum ./backend/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd backend && go mod download
COPY backend/ ./backend/
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd backend && CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/qwen2api .

# Stage 3: runtime image.
FROM debian:bookworm-slim
WORKDIR /app

ENV DEBIAN_FRONTEND=noninteractive \
    PORT=7860 \
    LOG_LEVEL=INFO \
    DATA_DIR=/app/data \
    LOGS_DIR=/app/logs \
    PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    wget \
    unzip \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libcups2 \
    libdbus-1-3 \
    libdrm2 \
    libgbm1 \
    libglib2.0-0 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libpangocairo-1.0-0 \
    libpulse0 \
    libx11-6 \
    libx11-xcb1 \
    libxcb1 \
    libxcomposite1 \
    libxdamage1 \
    libxext6 \
    libxfixes3 \
    libxkbcommon0 \
    libxrandr2 \
    libxshmfence1 \
    fonts-liberation \
    fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/*

COPY --from=backend-builder /out/qwen2api /usr/local/bin/qwen2api
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

RUN mkdir -p /app/data /app/logs /ms-playwright \
    && /usr/local/bin/qwen2api --install-browsers

EXPOSE 7860

HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=3 \
    CMD curl -fsS "http://127.0.0.1:${PORT:-7860}/healthz" || exit 1

CMD ["/usr/local/bin/qwen2api"]
