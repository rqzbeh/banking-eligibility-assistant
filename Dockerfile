# Multi-stage: Go backend + Vite React UI + Python agent gateway

# --- Stage 1: Go backend ---
FROM golang:1.22-alpine AS go-builder
WORKDIR /build
COPY backend/go.mod backend/go.sum* ./
RUN go mod download 2>/dev/null || true
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /banking-backend ./cmd/server

# --- Stage 2: Vite React build ---
FROM node:22-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# --- Stage 3: Runtime ---
FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=go-builder /banking-backend /app/backend-server
COPY --from=web-builder /web/dist /app/web/dist

RUN pip install --no-cache-dir \
    langchain langchain-openai langchain-core langgraph \
    openai httpx fastapi uvicorn

COPY agent/ /app/agent/
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

EXPOSE 8080 8501

ENV BACKEND_URL=http://localhost:8080 \
    BACKEND_PORT=8080 \
    UI_PORT=8501 \
    STATIC_DIR=/app/web/dist \
    OPENAI_BASE_URL=https://api.openai.com/v1 \
    LLM_MODEL=ag/gemini-3.6-flash-high \
    USE_RESPONSES_API=false \
    PYTHONPATH=/app/agent

ENTRYPOINT ["/app/docker-entrypoint.sh"]
