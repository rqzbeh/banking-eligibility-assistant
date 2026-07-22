import httpx

import server


BACKEND_URL = server.BACKEND_URL


def _customer(customer_id, national_id, name, occupation):
    return {
        "identity": {
            "customer_id": customer_id,
            "national_id": national_id,
            "name": name,
            "age": 34,
            "gender": "male",
            "occupation": occupation,
            "employment_type": "private",
            "customer_type": "real",
            "account_open_date": "1404/01/01",
            "is_existing": True,
        },
        "financial": {
            "customer_id": customer_id,
            "monthly_income": 120_000_000,
            "account_turnover_3m": 600_000_000,
            "account_turnover_12m": 2_000_000_000,
            "total_deposits": 800_000_000,
            "active_loans": 0,
            "total_loan_amount": 0,
            "installment_default": 0,
            "spending_pattern": "moderate",
            "payment_history": "excellent",
            "has_guarantor": True,
        },
        "risk": {
            "customer_id": customer_id,
            "risk_level": "low",
            "risk_score": 10,
            "reason": "lookup test",
            "is_cold_start": False,
        },
    }


def test_fallback_chat_finds_customer_by_name():
    reply = server.fallback_chat("اهلیت علی رضایی را برای دسته‌چک بررسی کن", thread_id="name-lookup")

    assert "علی رضایی" in reply
    assert "دسته‌چک" in reply


def test_fallback_chat_finds_customer_by_customer_id():
    reply = server.fallback_chat("شناسه C002 را برای دسته‌چک بررسی کن", thread_id="customer-id-lookup")

    assert "علی رضایی" in reply
    assert "دسته‌چک" in reply


def test_fallback_chat_asks_for_clarification_and_uses_thread_memory():
    duplicate = _customer("C902", "7654321098", "علی رضایی", "housewife")
    with httpx.Client(base_url=BACKEND_URL, timeout=10, trust_env=False) as client:
        client.delete("/api/rbci/customers/7654321098")
        created = client.post("/api/rbci/customers", json=duplicate)
        assert created.status_code == 201
        try:
            first = server.fallback_chat("اهلیت علی رضایی را بررسی کن", thread_id="ambiguous-name")
            assert "چند مشتری مشابه پیدا شد" in first
            assert "شغل مدیر" in first
            assert "شغل خانه‌دار" in first

            second = server.fallback_chat("مدیر", thread_id="ambiguous-name")
            assert "علی رضایی" in second
            assert "سطح ریسک" in second
        finally:
            client.delete("/api/rbci/customers/7654321098")
