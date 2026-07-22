<div dir="rtl" align="right">

# مستندات API — دستیار هوشمند بانکی

آدرس پایه: `http://localhost:8080`

---

## بررسی سلامت سرویس

### `GET /api/health`
**پاسخ:**
```json
{"status": "ok", "service": "banking-assistant-backend", "customer_store": "local-rbci"}
```

`customer_store` همیشه `local-rbci` است. PostgreSQL فقط persistence داخلی همین endpoint محلی است.

---

## سرویس اطلاعات هویتی

### `GET /api/identity?national_id={کد_ملی}`

**پارامترها:**
| پارامتر | نوع | الزامی | توضیح |
|---------|-----|--------|-------|
| national_id | string | بله | کد ملی ۱۰ رقمی |

**پاسخ موفق (200):**
```json
{
  "customer_id": "C001",
  "national_id": "0012345678",
  "name": "فاطمه احمدی",
  "age": 40,
  "gender": "female",
  "occupation": "housewife",
  "employment_type": "none",
  "customer_type": "real",
  "account_open_date": "1400/06/15",
  "is_existing": true
}
```

**خطاها:**
- `400` — کد ملی ارسال نشده
- `404` — مشتری یافت نشد

---

## سرویس اطلاعات مالی

### `GET /api/financial?customer_id={شناسه_مشتری}`

**پاسخ موفق (200):**
```json
{
  "customer_id": "C001",
  "monthly_income": 8000000,
  "account_turnover_3m": 30000000,
  "account_turnover_12m": 100000000,
  "total_deposits": 50000000,
  "active_loans": 0,
  "total_loan_amount": 0,
  "installment_default": 0,
  "spending_pattern": "conservative",
  "payment_history": "good",
  "has_guarantor": false
}
```

---

## سامانه RBCI (ارزیابی ریسک)

### `GET /api/rbci?customer_id={شناسه_مشتری}`

**پاسخ موفق (200):**
```json
{
  "customer_id": "C001",
  "risk_level": "medium",
  "risk_score": 55,
  "reason": "درآمد ثابت اندک، بدون سابقه وام",
  "is_cold_start": false
}
```

### `POST /api/rbci/cold-start`
ارزیابی ریسک اولیه برای افراد غیرمشتری

**بدنه درخواست:**
```json
{
  "name": "سارا نوروزی",
  "age": 30,
  "gender": "female",
  "occupation": "employee",
  "employment_type": "private",
  "approx_income": 20000000,
  "visit_purpose": "وام شخصی"
}
```

**مقادیر مجاز occupation:** `employee`, `self_employed`, `housewife`, `retired`, `unemployed`, `manager`, `student`

**مقادیر مجاز employment_type:** `government`, `private`, `freelance`, `none`

---

## endpoint محلی RBCI برای داده مشتری

این contract جای RBCI آنلاین را در محیط دمو می‌گیرد. در Docker، PostgreSQL adapter محلی همین endpoint است و موتور تطبیق هم همین داده‌ها را می‌خواند.

برای اتصال به RBCI آنلاین، همین pathها را برای UI نگه دارید و فقط adapter پشت آن‌ها را عوض کنید. اگر RBCI آنلاین عملیات write داشته باشد، `POST/PUT/DELETE` باید به RBCI push شود؛ اگر read-only باشد، همین مسیرها باید خطای روشن `405` یا `501` برگردانند.

### `GET /api/rbci/customers`

فهرست رکوردهای هویت، مالی و ریسک.

### `POST /api/rbci/customers`

افزودن مشتری جدید به endpoint محلی RBCI.

### `GET /api/rbci/customers/{national_id}`

خواندن یک رکورد با کد ملی ۱۰ رقمی.

### `PUT /api/rbci/customers/{national_id}`

ویرایش رکورد. `national_id` و `customer_id` رکورد موجود نباید عوض شوند.

### `DELETE /api/rbci/customers/{national_id}`

حذف رکورد از endpoint محلی RBCI. در PostgreSQL حذف پایدار است و seed با restart برنمی‌گردد.

**بدنه create/update:**
```json
{
  "identity": {
    "customer_id": "C006",
    "national_id": "1234567890",
    "name": "مشتری تست",
    "age": 34,
    "gender": "male",
    "occupation": "employee",
    "employment_type": "private",
    "customer_type": "real",
    "account_open_date": "1404/01/01",
    "is_existing": true
  },
  "financial": {
    "customer_id": "C006",
    "monthly_income": 40000000,
    "account_turnover_3m": 150000000,
    "account_turnover_12m": 600000000,
    "total_deposits": 0,
    "active_loans": 0,
    "total_loan_amount": 0,
    "installment_default": 0,
    "spending_pattern": "moderate",
    "payment_history": "good",
    "has_guarantor": false
  },
  "risk": {
    "customer_id": "C006",
    "risk_level": "low",
    "risk_score": 20,
    "reason": "ورودی محلی RBCI",
    "is_cold_start": false
  }
}
```

---

## محصولات بانکی

### `GET /api/products`

**پاسخ:** آرایه‌ای از محصولات
```json
[
  {
    "id": "P001",
    "name": "Personal Loan",
    "name_fa": "وام شخصی",
    "category": "loan",
    "description": "Personal loan up to 500M Toman",
    "description_fa": "وام شخصی تا سقف ۵۰۰ میلیون تومان"
  }
]
```

---

## بخشنامه‌ها

### `GET /api/circulars`
فهرست تمام بخشنامه‌ها

### `GET /api/circulars/by-product?product_id={شناسه_محصول}`
بخشنامه‌های مرتبط با یک محصول خاص

**پاسخ:**
```json
[
  {
    "id": "R001",
    "circular_ref": "BN-1404/123",
    "circular_ref_fa": "بخشنامه شماره ۱۴۰۴/۱۲۳",
    "product_id": "P001",
    "conditions": [
      {"field": "age", "operator": "gte", "value": 18},
      {"field": "monthly_income", "operator": "gte", "value": 10000000}
    ],
    "description_fa": "شرایط اهلیت وام شخصی..."
  }
]
```

---

## موتور تطبیق

### `POST /api/match`
تطبیق کامل مشتری موجود با تمام محصولات

**بدنه درخواست:**
```json
{
  "national_id": "0012345678",
  "include_default_warning": true
}
```

**پاسخ موفق (200):**
```json
{
  "customer_id": "C001",
  "national_id": "0012345678",
  "customer_name": "فاطمه احمدی",
  "is_existing": true,
  "is_cold_start": false,
  "risk_level": "medium",
  "risk_score": 55,
  "eligible_products": [
    {
      "product_id": "P005",
      "product_name_fa": "سپرده کوتاه‌مدت",
      "eligible": true,
      "reasons_fa": ["تمام شرایط رعایت شده است"],
      "circular_refs": ["BN-1404/127"],
      "score": 60
    }
  ],
  "ineligible_products": [
    {
      "product_id": "P001",
      "product_name_fa": "وام شخصی",
      "eligible": false,
      "reasons_fa": ["درآمد ماهانه باید حداقل ۱۰ میلیون باشد (فعلی: ۸ میلیون)"],
      "gaps": [
        {
          "field": "monthly_income",
          "current_value": "8000000",
          "required_value": "gte 10000000",
          "description_fa": "درآمد ماهانه باید حداقل ۱۰ میلیون باشد (فعلی: ۸ میلیون)"
        }
      ],
      "circular_refs": ["BN-1404/123"]
    }
  ],
  "personalized_offers": [...],
  "default_warning": {
    "current_risk_level": "medium",
    "potential_risk_level": "high",
    "consequences_fa": [
      "سطح ریسک از متوسط به بالا افزایش می‌یابد",
      "تمام درخواست‌های وام رد خواهد شد",
      "اقدامات حقوقی وصول مطالبات ممکن است آغاز شود"
    ]
  }
}
```

### `POST /api/match/cold-start`
تطبیق فرد غیرمشتری (بر مبنای اطلاعات خوداظهاری)

**بدنه درخواست:** مانند `cold-start` ریسک + فیلد `include_default_warning`

---

## کدهای خطا

| کد | توضیح |
|----|-------|
| 200 | موفق |
| 400 | پارامتر الزامی ارسال نشده یا نامعتبر |
| 409 | مشتری یا شناسه مشتری تکراری |
| 404 | مشتری / منبع یافت نشد |
| 503 | یکی از ورودی‌های محلی RBCI برای matching در دسترس نیست |
| 500 | خطای داخلی سرور |

</div>
