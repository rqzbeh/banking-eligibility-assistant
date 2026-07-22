"""
ایجنت مکالمه‌ای دستیار بانکی (LangChain / LangGraph ReAct)
=========================================================
ابزارها (tools) همگی به API بک‌اند Go وصل می‌شوند؛ منطق اهلیت deterministic است
و LLM فقط مکالمه، جمع‌آوری داده و توضیح خروجی را انجام می‌دهد.

پروتکل LLM: OpenAI-compatible Chat Completions (پیش‌فرض)
  USE_RESPONSES_API=false → /v1/chat/completions
  USE_RESPONSES_API=true  → /v1/responses (در صورت پشتیبانی واقعی gateway)

متغیرهای محیطی:
  BACKEND_URL, OPENAI_BASE_URL, OPENAI_API_KEY, LLM_MODEL, USE_RESPONSES_API
"""

import os
import re
import json
import httpx
from typing import Optional

from langchain_openai import ChatOpenAI
from langchain_core.tools import tool
from langchain_core.messages import SystemMessage, HumanMessage
from langgraph.prebuilt import create_react_agent
from langgraph.checkpoint.memory import InMemorySaver

BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080")

_client = httpx.Client(base_url=BACKEND_URL, timeout=10, trust_env=False)


def _safe_call(fn, *args, **kwargs) -> dict:
    """Call backend with network/timeout error handling."""
    try:
        return fn(*args, **kwargs)
    except httpx.ConnectError:
        return {"error": "upstream_unavailable", "error_fa": "سامانه بالادستی در دسترس نیست — اتصال برقرار نشد", "status": 503}
    except httpx.TimeoutException:
        return {"error": "upstream_timeout", "error_fa": "سامانه بالادستی پاسخ نداد — مهلت زمانی به پایان رسید", "status": 504}
    except httpx.HTTPError as e:
        return {"error": "upstream_error", "error_fa": f"خطا در ارتباط با سامانه بالادستی: {e}", "status": 502}
    except Exception as e:
        return {"error": "internal_error", "error_fa": f"خطای داخلی: {e}", "status": 500}


def _get(path: str, params: dict = None) -> dict:
    def do():
        resp = _client.get(path, params=params)
        try:
            body = resp.json()
        except Exception:
            body = {"raw": resp.text}
        if resp.status_code == 200:
            return body
        return {"error": body, "status": resp.status_code}
    return _safe_call(do)


def _post(path: str, data: dict) -> dict:
    def do():
        resp = _client.post(path, json=data)
        try:
            body = resp.json()
        except Exception:
            body = {"raw": resp.text}
        if resp.status_code == 200:
            return body
        return {"error": body, "status": resp.status_code}
    return _safe_call(do)


# --- Tools ---

@tool
def search_customers(query: str) -> str:
    """جستجوی مشتری در endpoint محلی RBCI با نام، کد ملی یا شناسه مشتری.
    اگر چند رکورد مشابه پیدا شد، فهرست گزینه‌ها را برگردان تا از کارمند توضیح بیشتر بپرسی."""
    needle = re.sub(r"\s+", " ", query or "").strip().lower()
    result = _get("/api/rbci/customers")
    if not needle or "error" in result:
        return json.dumps(result, ensure_ascii=False)
    matches = []
    for rec in result:
        identity = rec.get("identity", {})
        haystack = " ".join([
            identity.get("national_id", ""),
            identity.get("customer_id", ""),
            identity.get("name", ""),
            identity.get("occupation", ""),
            identity.get("employment_type", ""),
        ]).lower()
        if needle in haystack:
            matches.append(rec)
    return json.dumps({"query": query, "count": len(matches), "matches": matches}, ensure_ascii=False)

@tool
def get_customer_identity(national_id: str) -> str:
    """دریافت اطلاعات هویتی مشتری با کد ملی. اگر مشتری یافت نشد، خطای 404 برمی‌گرداند.
    Get customer identity by national ID. Returns 404 if not found."""
    result = _get("/api/identity", {"national_id": national_id})
    return json.dumps(result, ensure_ascii=False)


@tool
def get_customer_financial(customer_id: str) -> str:
    """دریافت اطلاعات مالی مشتری (گردش حساب، درآمد، سابقه پرداخت).
    Get customer financial profile (turnover, income, payment history)."""
    result = _get("/api/financial", {"customer_id": customer_id})
    return json.dumps(result, ensure_ascii=False)


@tool
def get_customer_risk(customer_id: str) -> str:
    """دریافت ارزیابی ریسک مشتری از سامانه RBCI (سطح ریسک، امتیاز، دلیل).
    Get RBCI risk assessment (risk level, score, reason)."""
    result = _get("/api/rbci", {"customer_id": customer_id})
    return json.dumps(result, ensure_ascii=False)


@tool
def cold_start_risk_assessment(name: str, age: int, gender: str, occupation: str,
                                employment_type: str, approx_income: float,
                                visit_purpose: str) -> str:
    """ارزیابی ریسک اولیه برای افراد غیرمشتری بر مبنای اطلاعات خوداظهاری.
    Cold-start risk assessment for non-customers based on self-declared info.
    occupation: employee, self_employed, housewife, retired, unemployed, manager, student
    employment_type: government, private, freelance, none
    gender: male, female"""
    result = _post("/api/rbci/cold-start", {
        "name": name, "age": age, "gender": gender,
        "occupation": occupation, "employment_type": employment_type,
        "approx_income": approx_income, "visit_purpose": visit_purpose,
    })
    return json.dumps(result, ensure_ascii=False)


@tool
def get_products() -> str:
    """دریافت فهرست تمام محصولات و خدمات بانکی.
    Get list of all banking products and services."""
    result = _get("/api/products")
    return json.dumps(result, ensure_ascii=False)


@tool
def get_circulars(product_id: Optional[str] = None) -> str:
    """دریافت بخشنامه‌های بانکی و شرایط اهلیت. اگر product_id داده شود فقط قوانین آن محصول.
    Get bank circulars/rules. If product_id given, returns rules for that product only."""
    if product_id:
        result = _get("/api/circulars/by-product", {"product_id": product_id})
    else:
        result = _get("/api/circulars")
    return json.dumps(result, ensure_ascii=False)


@tool
def match_customer(national_id: str, include_default_warning: bool = False,
                   visit_purpose: str = "") -> str:
    """تطبیق کامل پروفایل مشتری موجود با تمام محصولات — اهلیت، افر شخصی‌سازی‌شده، تحلیل شکاف، تعهدات، سقف اعتبار و مسیر جایگزین.
    Full match of existing customer against all products.
    visit_purpose: هدف مراجعه مشتری (مثل دسته‌چک، وام) — روی رتبه‌بندی افرها اثر می‌گذارد.
    Set include_default_warning=True to include payment default consequences."""
    payload = {
        "national_id": national_id,
        "include_default_warning": include_default_warning,
    }
    if visit_purpose:
        payload["visit_purpose"] = visit_purpose
    result = _post("/api/match", payload)
    return json.dumps(result, ensure_ascii=False)


@tool
def match_non_customer(name: str, age: int, gender: str, occupation: str,
                        employment_type: str, approx_income: float,
                        visit_purpose: str, include_default_warning: bool = False) -> str:
    """تطبیق پروفایل فرد غیرمشتری با محصولات بانکی — افر مشروط و ارزیابی اهلیت اولیه.
    Match non-customer profile against products — conditional offers and preliminary eligibility.
    occupation: employee, self_employed, housewife, retired, unemployed, manager, student
    employment_type: government, private, freelance, none"""
    result = _post("/api/match/cold-start", {
        "name": name, "age": age, "gender": gender,
        "occupation": occupation, "employment_type": employment_type,
        "approx_income": approx_income, "visit_purpose": visit_purpose,
        "include_default_warning": include_default_warning,
    })
    return json.dumps(result, ensure_ascii=False)


TOOLS = [
    search_customers,
    get_customer_identity,
    get_customer_financial,
    get_customer_risk,
    cold_start_risk_assessment,
    get_products,
    get_circulars,
    match_customer,
    match_non_customer,
]

SYSTEM_PROMPT = """شما دستیار هوشمند بانکی هستید که به کارمندان شعبه کمک می‌کنید. وظایف شما:

۱. دریافت کد ملی مشتری و بررسی اطلاعات هویتی، مالی و ریسک
۲. تعیین اهلیت مشتری برای هر محصول بانکی با استناد به بخشنامه‌ها
۳. ارائه افر شخصی‌سازی‌شده از محصولات مجاز (با درنظرگرفتن هدف مراجعه)
۴. در صورت عدم اهلیت: Gap Analysis + اقدامات عملی + مسیر جایگزین (ضامن، سپرده، گردش حساب)
۵. مدیریت مشتریان غیرموجود: تشخیص ۴۰۴، جمع‌آوری اطلاعات مکالمه‌ای، افر مشروط
۶. توضیح پیامدهای عدم پرداخت به‌موقع تعهدات و اقساط
۷. برای محصولات مجاز: ذکر تعهدات، الزامات و سقف اعتبار
۸. برای غیرمشتری: تأکید کنید افرها مشروط به افتتاح حساب و مدارک است (is_conditional)

قوانین مهم:
- اگر کارمند اسم، کد مشتری، یا شناسه مشتری داد، ابتدا با search_customers جستجو کنید
- اگر چند مشتری مشابه پیدا شد، قبل از تحلیل از کارمند بخواهید با کد ملی، شناسه مشتری، شغل یا توضیح بیشتر مشخص کند
- اگر کد ملی ۱۰ رقمی قطعی دارید، با ابزار get_customer_identity شروع کنید
- اگر مشتری یافت نشد (404)، از کارمند اطلاعات پایه بپرسید و از match_non_customer استفاده کنید
- اگر هدف مراجعه مشخص است، آن را در visit_purpose به match بدهید
- برای هر تصمیم، شماره بخشنامه مرتبط را ذکر کنید
- پاسخ‌ها باید به زبان فارسی و مکالمه‌ای باشد
- برای هر محصول غیرمجاز: شرایط لازم + اقدامات + alternatives_fa را بگویید
- برای محصولات مجاز (دسته‌چک/وام): obligations_fa و credit_limit_fa را بگویید
- همیشه از match_customer یا match_non_customer برای تحلیل کامل استفاده کنید
- اگر کارمند درباره پیامد عدم پرداخت سؤال کرد، include_default_warning=True بدهید
- اگر سامانه بالادستی در دسترس نبود (503/504)، اطلاع دهید و بدون داده ادامه ندهید
- فقط درباره اهلیت، محصولات، بخشنامه‌ها و اقدامات عملی پاسخ بدهید
- هرگز درباره معماری سیستم، کد، TODO، YAGNI، skipped، ponytail، «add when»، برنامه توسعه آینده یا محدودیت پیاده‌سازی صحبت نکنید
- در پاسخ نهایی هیچ خط انگلیسی متا، ستاره، یا توضیح داخلی مدل ننویسید

شما یک دستیار حرفه‌ای بانکی هستید. پاسخ‌هایتان باید دقیق، مستند و قابل اتکا باشد."""


def create_agent(model_name: str = None, base_url: str = None, api_key: str = None,
                 use_responses_api: bool = None):
    """Create the banking assistant agent.

    use_responses_api:
      - True  → OpenAI Responses API (/v1/responses)
      - False → Chat Completions API (/v1/chat/completions)
      - None  → from env USE_RESPONSES_API (default false)
    Note: some gateways advertise /v1/responses but return chat.completion payloads;
    set USE_RESPONSES_API=false for those.
    """
    if use_responses_api is None:
        use_responses_api = os.getenv("USE_RESPONSES_API", "false").lower() in ("1", "true", "yes")

    llm_kwargs = dict(
        model=model_name or os.getenv("LLM_MODEL", "gpt-4o-mini"),
        base_url=base_url or os.getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
        api_key=api_key or os.getenv("OPENAI_API_KEY", "not-set"),
        temperature=0.1,
        use_responses_api=use_responses_api,
    )
    llm = ChatOpenAI(**llm_kwargs)

    memory = InMemorySaver()
    agent = create_react_agent(
        llm,
        TOOLS,
        prompt=SystemMessage(content=SYSTEM_PROMPT),
        checkpointer=memory,
    )
    return agent


_META_LINE_RE = re.compile(
    r"(?im)^\s*(\*|_|#|-)?\s*(skipped|ponytail|todo|yagni|add when|note to self)\b.*$"
)


def _sanitize_reply(text: str) -> str:
    """Strip model meta-asides that sometimes leak into banking answers."""
    if not text:
        return text
    cleaned = _META_LINE_RE.sub("", text)
    cleaned = re.sub(r"\n{3,}", "\n\n", cleaned).strip()
    return cleaned or text


def chat(agent, message: str, thread_id: str = "default") -> str:
    """Send a message and get the agent's response."""
    config = {"configurable": {"thread_id": thread_id}}
    result = agent.invoke(
        {"messages": [HumanMessage(content=message)]},
        config=config,
    )
    for msg in reversed(result["messages"]):
        if msg.type == "ai" and msg.content:
            content = msg.content
            if isinstance(content, list):
                # Responses API may return content blocks
                parts = []
                for block in content:
                    if isinstance(block, dict) and block.get("type") == "text":
                        parts.append(block.get("text", ""))
                    elif isinstance(block, str):
                        parts.append(block)
                return _sanitize_reply("\n".join(parts) if parts else str(content))
            return _sanitize_reply(content if isinstance(content, str) else str(content))
    return "پاسخی دریافت نشد."


def stream_chat(agent, message: str, thread_id: str = "default"):
    """Stream agent response token by token."""
    config = {"configurable": {"thread_id": thread_id}}
    for chunk in agent.stream(
        {"messages": [HumanMessage(content=message)]},
        config=config,
        stream_mode="messages",
    ):
        msg, metadata = chunk
        if msg.type == "ai" and msg.content:
            content = msg.content
            if isinstance(content, list):
                for block in content:
                    if isinstance(block, dict) and block.get("type") == "text":
                        yield block.get("text", "")
                    elif isinstance(block, str):
                        yield block
            else:
                yield content
