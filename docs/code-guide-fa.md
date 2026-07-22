# راهنمای کد — دستیار هوشمند اهلیت بانکی

این سند ساختار کد را برای داوران و توسعه‌دهندگان توضیح می‌دهد.

## معماری کلی

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ React SPA       │────▶│ Python Agent     │────▶│ Go Backend :8080│
│  (web/)         │     │ (agent/agent.py) │     │ Mock APIs+Engine│
└─────────────────┘     └────────┬─────────┘     └────────▲────────┘
                                 │ Chat Completions       │
                                 ▼                        │
                        OpenAI-compatible LLM             │
                        (OPENAI_BASE_URL)                 │
                                                          │
        UI «بررسی سریع» ──────────────────────────────────┘
```

- **منطق اهلیت قطعی است** (Go engine) — LLM فقط مکالمه و توضیح می‌کند.
- **Chat Completions** پیش‌فرض است (`USE_RESPONSES_API=false`).

## ساختار پوشه‌ها

| مسیر | نقش |
|------|-----|
| `backend/cmd/server/main.go` | نقطه ورود HTTP، ثبت مسیرها، CORS |
| `backend/internal/models/` | قرارداد JSON ورودی/خروجی |
| `backend/internal/data/` | داده‌های Mock (مشتری، محصول، بخشنامه) |
| `backend/internal/engine/` | موتور اهلیت، gap، افر، cold-start |
| `backend/internal/handlers/` | لایه HTTP و اعتبارسنجی |
| `agent/agent.py` | ایجنت LangChain با tools به بک‌اند |
| `agent/server.py` | Gateway: SPA + پروکسی API + چت agent |
| `web/` | رابط وب RTL کارمند شعبه |
| `docs/` | مستندات فارسی + OpenAPI + Postman + دمو |

## بک‌اند Go — جریان تطبیق

### مشتری موجود — `POST /api/match`

1. `ValidateNationalID` — کد ملی ۱۰ رقمی
2. خواندن Identity / Financial / RBCI (در صورت نبود → `503` + `upstream_errors`)
3. `BuildProfile` — تخت‌سازی فیلدها برای ارزیابی شرط
4. `MatchAllProducts` — برای هر محصول:
   - همه شروط بخشنامه `EvaluateCondition`
   - مجاز → `ObligationsFa` + `CreditLimitFa` + `Score`
   - غیرمجاز → `Gaps` + `AdviceFa` + `AlternativesFa`
5. `visitPurposeBoost` — افزایش امتیاز بر اساس هدف مراجعه
6. `PersonalizedOffers` — ۳ افر برتر
7. اختیاری: `GenerateDefaultWarning`

### غیرمشتری — `POST /api/match/cold-start`

1. اعتبارسنجی خوداظهاری (نام، سن ۱۵–۱۰۰، شغل، جنسیت، درآمد ≥۰)
2. `NormalizeOccupation` — نگاشت فارسی→انگلیسی (`مدیر`→`manager`)
3. `ColdStartRisk` — امتیاز ریسک clamped بین ۵–۹۵
4. `BuildProfileFromColdStart` با `IsColdStart=true`
5. محصولات مجاز با `IsConditional=true` و `ConditionsFa`
6. `NotesFa` — یادداشت سیستمی افر مشروط

## عملگرهای شرط بخشنامه

| Operator | معنی |
|----------|------|
| `eq` / `neq` | برابری / نابرابری |
| `gt` / `gte` / `lt` / `lte` | مقایسه‌های عددی |
| `in` / `not_in` | عضویت در لیست |

فیلد گم‌شده → شرط fail + reason فارسی.

## سناریوهای PDF و داده Mock

| کد ملی | مشتری | انتظار کلیدی |
|--------|--------|---------------|
| `0012345678` | خانه‌دار ۴۰ساله | وام غیرمجاز + alternatives؛ سپرده مجاز |
| `0023456789` | مدیر پردرآمد | دسته‌چک/وام مجاز + obligations + credit_limit |
| `0034567890` | کارمند با معوقه | دسته‌چک غیرمجاز + advice؛ کارت اعتباری مجاز |
| — | cold-start | افر مشروط + notes_fa |

## ایجنت Python

فایل: `agent/agent.py`

- Tools: `get_customer_identity`, `get_customer_financial`, `get_customer_risk`,
  `cold_start_risk_assessment`, `get_products`, `get_circulars`,
  `match_customer`, `match_non_customer`
- `_safe_call`: خطای بالادستی را با `error`/`error_fa` برمی‌گرداند
- LLM: `ChatOpenAI(..., use_responses_api=False)` مگر `USE_RESPONSES_API=true`
- System prompt فارسی: ارجاع بخشنامه، gap، تعهدات، افر مشروط
- Memory: `InMemorySaver` + `thread_id`
- `_sanitize_reply`: حذف خطوط متای مدل

## رابط کاربری

مسیر: `web/` (React) — سرو از `agent/server.py`

- حالت «بررسی سریع»: `POST /api/match` مستقیم
- حالت «دستیار هوشمند»: چت با agent
- رندر: offers، eligible (تعهدات/سقف/مشروط)، ineligible (gap/advice/alternatives)، default_warning، notes_fa

## متغیرهای محیطی

| متغیر | پیش‌فرض | توضیح |
|-------|---------|--------|
| `BACKEND_URL` | `http://localhost:8080` | آدرس بک‌اند |
| `OPENAI_BASE_URL` | — | endpoint سازگار OpenAI |
| `OPENAI_API_KEY` | — | کلید API |
| `LLM_MODEL` | — | نام مدل |
| `USE_RESPONSES_API` | `false` | فقط در صورت پشتیبانی واقعی gateway |
| `STATIC_DIR` | — | مسیر `web/dist` برای SPA |

## تست‌ها

```bash
# Go
cd backend && go test ./... -count=1

# Python (بک‌اند را در صورت نیاز خودش بالا می‌آورد)
cd agent && pytest -q
```

پوشش: عملگرهای شرط، سناریوهای PDF، NID نامعتبر، شغل فارسی،
افر مشروط، boost هدف مراجعه، alternatives، obligations، default warning.

## استقرار Docker

```bash
docker build -t banking-assistant .
docker run -d --name banking-assistant \
  -p 9080:8080 -p 9501:8501 \
  -e OPENAI_BASE_URL=... -e OPENAI_API_KEY=... -e LLM_MODEL=... \
  -e USE_RESPONSES_API=false \
  banking-assistant
```

entrypoint: backend Go + gateway (SPA React + agent).
