"""
Gateway HTTP برای ایجنت + سرو SPA (React) + پروکسی API بک‌اند Go

مسیرها:
  POST /api/agent/chat     → LangChain agent
  GET  /api/agent/health
  /api/*                   → پروکسی به Go backend (BACKEND_URL)
  /*                       → فایل‌های build شده Vite (STATIC_DIR)
"""
from __future__ import annotations

import os
import re
import uuid
from pathlib import Path

import httpx
from fastapi import FastAPI, HTTPException, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, JSONResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel, Field

from agent import chat, create_agent

BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080").rstrip("/")
STATIC_DIR = Path(
    os.getenv("STATIC_DIR", str(Path(__file__).resolve().parent.parent / "web" / "dist"))
)

app = FastAPI(title="Banking Assistant Gateway", version="1.0.0")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

_agent = None
_http: httpx.AsyncClient | None = None


def get_agent():
    global _agent
    if _agent is None:
        _agent = create_agent()
    return _agent


def get_http() -> httpx.AsyncClient:
    global _http
    if _http is None:
        _http = httpx.AsyncClient(base_url=BACKEND_URL, timeout=30.0, trust_env=False)
    return _http


def fallback_chat(message: str) -> str:
    national_id = next((m.group(0) for m in re.finditer(r"\d{10}", message)), "")
    if not national_id:
        return "برای بررسی اهلیت، کد ملی ۱۰ رقمی مشتری را وارد کنید."

    visit_purpose = ""
    if any(k in message for k in ("دسته‌چک", "دسته چک", "چک")):
        visit_purpose = "دسته‌چک"
    elif "مسکن" in message:
        visit_purpose = "وام مسکن"
    elif any(k in message for k in ("وام", "تسهیلات")):
        visit_purpose = "وام"
    elif "سپرده" in message:
        visit_purpose = "سپرده"

    with httpx.Client(base_url=BACKEND_URL, timeout=10.0, trust_env=False) as client:
        resp = client.post(
            "/api/match",
            json={
                "national_id": national_id,
                "visit_purpose": visit_purpose,
                "include_default_warning": "عدم پرداخت" in message or "معوق" in message,
            },
        )

    if resp.status_code == 404:
        return "این کد ملی در بانک یافت نشد. برای مسیر غیرمشتری، نام، سن، شغل، نوع اشتغال، درآمد تقریبی و هدف مراجعه را وارد کنید."
    if resp.status_code != 200:
        return "در حال حاضر سامانه بالادستی پاسخ معتبر برنگرداند. کمی بعد دوباره تلاش کنید."

    data = resp.json()
    wanted = visit_purpose or "محصولات"
    products = data.get("eligible_products", []) + data.get("ineligible_products", [])
    if visit_purpose:
        products = [p for p in products if visit_purpose in p.get("product_name_fa", "")] or products
    lines = [
        f"نتیجه بررسی {wanted} برای {data.get('customer_name', national_id)}:",
        f"سطح ریسک: {data.get('risk_level')}؛ امتیاز: {data.get('risk_score')}",
    ]
    for product in products[:3]:
        status = "مجاز" if product.get("eligible") else "غیرمجاز"
        refs = "، ".join(product.get("circular_refs") or [])
        lines.append(f"{product.get('product_name_fa')}: {status}" + (f" بر اساس {refs}" if refs else ""))
        for reason in product.get("reasons_fa") or []:
            lines.append(f"دلیل: {reason}")
        for gap in product.get("gaps") or []:
            lines.append(f"شرط لازم: {gap.get('description_fa')}")
        for advice in product.get("advice_fa") or []:
            lines.append(f"اقدام پیشنهادی: {advice}")
        if product.get("credit_limit_fa"):
            lines.append(product["credit_limit_fa"])
        for obligation in product.get("obligations_fa") or []:
            lines.append(f"تعهد: {obligation}")
    return "\n".join(lines)


class ChatRequest(BaseModel):
    message: str = Field(..., min_length=1, max_length=8000)
    thread_id: str | None = None


class ChatResponse(BaseModel):
    reply: str
    thread_id: str


@app.on_event("shutdown")
async def _shutdown():
    global _http
    if _http is not None:
        await _http.aclose()
        _http = None


@app.get("/api/agent/health")
def agent_health():
    return {
        "status": "ok",
        "service": "banking-assistant-agent",
        "model": os.getenv("LLM_MODEL", ""),
        "use_responses_api": os.getenv("USE_RESPONSES_API", "false"),
        "backend_url": BACKEND_URL,
    }


@app.post("/api/agent/chat", response_model=ChatResponse)
def agent_chat(req: ChatRequest):
    msg = req.message.strip()
    if not msg:
        raise HTTPException(400, detail="message is required")
    thread_id = req.thread_id or str(uuid.uuid4())
    try:
        reply = chat(get_agent(), msg, thread_id=thread_id)
    except Exception as e:
        reply = fallback_chat(msg)
    return ChatResponse(reply=reply, thread_id=thread_id)


PROXY_PREFIXES = (
    "/api/health",
    "/api/identity",
    "/api/customers",
    "/api/financial",
    "/api/rbci",
    "/api/products",
    "/api/circulars",
    "/api/match",
)


@app.api_route("/api/{path:path}", methods=["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"])
async def proxy_backend(path: str, request: Request):
    full = f"/api/{path}"
    if full.startswith("/api/agent"):
        raise HTTPException(404, detail="not found")
    if not any(full.startswith(p) for p in PROXY_PREFIXES):
        raise HTTPException(404, detail="unknown api route")

    client = get_http()
    body = await request.body()
    headers = {
        k: v
        for k, v in request.headers.items()
        if k.lower() not in {"host", "content-length", "connection", "transfer-encoding"}
    }
    try:
        upstream = await client.request(
            request.method,
            full,
            params=request.query_params,
            content=body,
            headers=headers,
        )
    except httpx.RequestError as e:
        return JSONResponse(
            status_code=503,
            content={
                "error": "backend_unavailable",
                "error_fa": "بک‌اند در دسترس نیست",
                "detail": str(e),
            },
        )

    excluded = {"content-encoding", "transfer-encoding", "connection"}
    resp_headers = {k: v for k, v in upstream.headers.items() if k.lower() not in excluded}
    return Response(
        content=upstream.content,
        status_code=upstream.status_code,
        headers=resp_headers,
        media_type=upstream.headers.get("content-type"),
    )


if STATIC_DIR.is_dir():
    assets = STATIC_DIR / "assets"
    if assets.is_dir():
        app.mount("/assets", StaticFiles(directory=str(assets)), name="assets")

    @app.get("/{full_path:path}")
    async def spa(full_path: str):
        if full_path:
            candidate = (STATIC_DIR / full_path).resolve()
            try:
                candidate.relative_to(STATIC_DIR.resolve())
            except Exception:
                raise HTTPException(404, detail="not found")
            if candidate.is_file():
                return FileResponse(candidate)
        index = STATIC_DIR / "index.html"
        if index.is_file():
            return FileResponse(index)
        raise HTTPException(404, detail="UI not built")
else:

    @app.get("/")
    def no_ui():
        return {"error": "UI dist not found", "path": str(STATIC_DIR)}
