#!/bin/sh
set -e

echo "Starting Go backend on port ${BACKEND_PORT:-8080}..."
# nohup so backend survives exec of uvicorn (PID 1 replacement)
nohup /app/backend-server >/tmp/backend.log 2>&1 &
BACKEND_PID=$!

for i in $(seq 1 40); do
    if curl -sf "http://localhost:${BACKEND_PORT:-8080}/api/health" >/dev/null 2>&1; then
        echo "Backend is ready (pid $BACKEND_PID)."
        break
    fi
    sleep 0.25
done

if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
    echo "Backend failed to start:"
    cat /tmp/backend.log || true
    exit 1
fi

echo "Starting React UI + agent gateway on port ${UI_PORT:-8501}..."
cd /app/agent
exec python -m uvicorn server:app \
    --host 0.0.0.0 \
    --port "${UI_PORT:-8501}" \
    --log-level info
