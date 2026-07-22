# دستیار هوشمند بانکی — تعیین اهلیت و پیشنهاد محصولات

## توضیح پروژه

این سامانه یک دستیار هوشمند بانکی است که به کارمندان شعبه کمک می‌کند تا:

- **اهلیت مشتری** را برای هر محصول بانکی تعیین کنند
- **افر شخصی‌سازی‌شده** از محصولات مناسب ارائه دهند
- **تحلیل شکاف (Gap Analysis)** انجام دهند — اگر مشتری واجد شرایط نیست، دقیقاً چه باید تغییر کند
- **مشتریان غیرموجود** را از طریق مکالمه/فرم شناسایی و ارزیابی کنند
- **پیامدهای عدم پرداخت** تعهدات را توضیح دهند

## معماری سیستم

```
┌─────────────────┐   HTTP    ┌──────────────────────────┐
│  React SPA      │◄─────────►│  Python Agent Gateway    │
│  (web/ → dist)  │           │  FastAPI + LangChain     │
└─────────────────┘           └────────────┬─────────────┘
                                           │ HTTP
                              ┌────────────▼─────────────┐
                              │  Go Backend (net/http)   │
                              │  ├─ /api/identity        │
                              │  ├─ /api/financial       │
                              │  ├─ /api/rbci            │
                              │  ├─ /api/products        │
                              │  ├─ /api/circulars       │
                              │  └─ /api/match           │
                              └──────────────────────────┘
```

### اجزای سیستم

| جزء | زبان | توضیح |
|-----|------|-------|
| بک‌اند (Mock API + موتور قواعد) | Go | سرویس‌های هویتی، مالی، RBCI، محصولات، بخشنامه‌ها و موتور تطبیق |
| ایجنت هوشمند | Python (LangChain / LangGraph) | ارکستراسیون ابزارها، مدیریت مکالمه، حافظه جلسه |
| رابط کاربری | React (TypeScript) | رابط فارسی RTL برای کارمند شعبه — سرو از gateway |
| مدل زبانی | OpenAI-compatible | سازگار با Ollama، vLLM، OpenAI و سایر سرویس‌ها |

## نصب و اجرا

### روش ۱: Docker (توصیه‌شده)

```bash
docker build -t banking-assistant .
docker run -p 8080:8080 -p 8501:8501 \
  -e OPENAI_BASE_URL=http://your-llm-server/v1 \
  -e OPENAI_API_KEY=your-key \
  -e LLM_MODEL=gpt-4o-mini \
  -e USE_RESPONSES_API=false \
  banking-assistant

# رابط کاربری: http://localhost:8501
# API مستقیم:   http://localhost:8080
```

### روش ۲: اجرای مستقیم

**پیش‌نیازها:** Go 1.22+ · Python 3.12+ · Node 20+

```bash
# ۱. بک‌اند Go
cd backend && go run ./cmd/server
# → :8080

# ۲. ساخت رابط
cd web && npm ci && npm run build

# ۳. وابستگی‌های Python
pip install langchain langchain-openai langchain-core langgraph openai httpx fastapi uvicorn

# ۴. متغیرهای محیطی
export BACKEND_URL=http://localhost:8080
export OPENAI_BASE_URL=http://localhost:11434/v1   # مثال: Ollama
export OPENAI_API_KEY=not-needed-for-ollama
export LLM_MODEL=llama3.1
export USE_RESPONSES_API=false
export STATIC_DIR=$PWD/web/dist

# ۵. gateway (SPA + agent + پروکسی API)
cd agent && PYTHONPATH=. uvicorn server:app --host 0.0.0.0 --port 8501
# → http://localhost:8501
```

### اجرا با Ollama (مدل محلی)

```bash
ollama pull llama3.1
ollama serve

export OPENAI_BASE_URL=http://localhost:11434/v1
export LLM_MODEL=llama3.1
export OPENAI_API_KEY=not-needed
export USE_RESPONSES_API=false
```

## سناریوهای نمونه

### سناریو ۱: خانم خانه‌دار ۴۰ ساله
**کد ملی:** `0012345678` (فاطمه احمدی)

- درآمد ماهانه: ۸ میلیون تومان · سطح ریسک: متوسط
- **نتیجه:** غیرمجاز برای وام شخصی (نیاز به افزایش درآمد به ۱۰ میلیون) و دسته‌چک؛ مجاز برای سپرده و خدمات پایه؛ Gap + مسیر جایگزین (ضامن/سپرده/گردش)

### سناریو ۲: مدیر با درآمد بالا
**کد ملی:** `0023456789` (علی رضایی)

- درآمد ماهانه: ۱۲۰ میلیون تومان · سطح ریسک: کم
- **نتیجه:** مجاز برای دسته‌چک، وام‌ها و کارت اعتباری — با `obligations_fa` و `credit_limit_fa`

### سناریو ۳: کارمند با اقساط معوق
**کد ملی:** `0034567890` (محمد حسینی)

- درآمد: ۳۵ میلیون · ۲ قسط معوق
- **نتیجه:** غیرمجاز برای دسته‌چک/وام شخصی؛ مجاز برای کارت اعتباری (حداکثر ۲ معوقه)

### سناریو ۴: مشتری غیرموجود
**هر کد ملی خارج از نمونه** → `404` و مسیر cold-start با افر **مشروط** (`is_conditional`)

### سناریو ۵: پیامد عدم پرداخت
گزینه «هشدار عدم پرداخت» را فعال کنید:
- ریسک کم → متوسط، تعلیق دسته‌چک
- ریسک متوسط → بالا، رد وام‌ها، اقدام حقوقی
- ریسک بالا → محدودیت کامل خدمات

## اجرای تست‌ها

```bash
# بک‌اند Go
cd backend && go test ./... -count=1

# Python (بک‌اند را در صورت نیاز خودش بالا می‌آورد)
cd agent && pytest -q
```

## API و مستندات

| سند | مسیر |
|-----|------|
| OpenAPI / Swagger | [docs/openapi.yaml](docs/openapi.yaml) |
| Postman Collection | [docs/postman_collection.json](docs/postman_collection.json) |
| شرح فارسی API | [docs/api-docs.md](docs/api-docs.md) |
| نگاشت محصول ↔ اهلیت | [docs/product-eligibility-mapping.md](docs/product-eligibility-mapping.md) |
| گزارش معماری | [docs/architecture-report.md](docs/architecture-report.md) |
| راهنمای کد | [docs/code-guide-fa.md](docs/code-guide-fa.md) |
| اسکریپت دمو | [docs/demo-script.md](docs/demo-script.md) |

### نقاط پایانی اصلی

```
GET  /api/health
GET  /api/identity?national_id=0012345678
GET  /api/financial?customer_id=C001
GET  /api/rbci?customer_id=C001
POST /api/rbci/cold-start
GET  /api/products
GET  /api/circulars
GET  /api/circulars/by-product?product_id=P001
POST /api/match
POST /api/match/cold-start
```

Gateway (پورت UI):
```
POST /api/agent/chat   { "message": "...", "thread_id": "..." }
/*  →  SPA استاتیک (web/dist)
```

## ساختار پروژه

```
├── backend/                 # بک‌اند Go — Mock API + موتور قواعد
│   ├── cmd/server/
│   └── internal/{models,data,engine,handlers}/
├── agent/                   # ایجنت + gateway FastAPI
│   ├── agent.py
│   ├── server.py
│   └── test_*.py
├── web/                     # رابط React (TypeScript)
├── docs/                    # مستندات، OpenAPI، Postman، دمو
├── Dockerfile
└── README.md
```

## روش استخراج قواعد از بخشنامه‌ها

قواعد بخشنامه‌ها به صورت ساختاریافته (JSON) مدل‌سازی شده‌اند. هر بخشنامه شامل:

1. **شناسه بخشنامه** (مثال: BN-1404/123)
2. **شناسه محصول** مرتبط
3. **شرایط اهلیت** به‌صورت لیستی از قواعد شرطی (فیلد / عملگر / آستانه)

```json
{
  "circular_ref": "BN-1404/123",
  "product_id": "P001",
  "conditions": [
    {"field": "age", "operator": "gte", "value": 18},
    {"field": "monthly_income", "operator": "gte", "value": 10000000},
    {"field": "risk_level", "operator": "in", "value": ["low", "medium"]}
  ]
}
```

موتور تطبیق هر شرط را با پروفایل مشتری مقایسه می‌کند. در صورت عدم تطابق:
- **دلیل رد** با ارجاع به بخشنامه
- **تحلیل شکاف** + اقدامات عملی + مسیر جایگزین

## LLM

پروتکل پیش‌فرض: **Chat Completions** (`USE_RESPONSES_API=false`).

## رابط کاربری

- مسیر: `web/` — رابط وب RTL برای کارمند شعبه
- ساخت: `cd web && npm ci && npm run build` → `web/dist`
- سرو: gateway در `agent/server.py` (SPA + پروکسی API + چت agent) روی پورت 8501
- توسعه: `cd web && npm run dev` (پروکسی به backend :8080 و agent :8501)

## مجوز

این پروژه برای چالش ICTChallenge / DATA توسعه یافته است.
