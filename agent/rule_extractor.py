"""LLM-assisted circular rule extraction with deterministic validation."""

from __future__ import annotations

import argparse
import json
import os
import re
from pathlib import Path

from langchain_core.messages import HumanMessage, SystemMessage
from langchain_openai import ChatOpenAI

from agent import DEFAULT_LLM_MODEL


PRODUCTS = {
    "P001": "وام شخصی",
    "P002": "وام مسکن",
    "P003": "دسته‌چک",
    "P004": "کارت اعتباری",
    "P005": "سپرده کوتاه‌مدت",
    "P006": "سپرده بلندمدت",
    "P007": "وام کسب‌وکار",
    "P008": "بانکداری اینترنتی",
    "P009": "پیامک بانکی",
    "P010": "وام ازدواج",
}
ALLOWED_FIELDS = {
    "age",
    "gender",
    "occupation",
    "employment_type",
    "customer_type",
    "monthly_income",
    "account_turnover_3m",
    "account_turnover_12m",
    "total_deposits",
    "active_loans",
    "total_loan_amount",
    "installment_default",
    "spending_pattern",
    "payment_history",
    "has_guarantor",
    "risk_level",
    "risk_score",
}
NUMERIC_FIELDS = {
    "age",
    "monthly_income",
    "account_turnover_3m",
    "account_turnover_12m",
    "total_deposits",
    "active_loans",
    "total_loan_amount",
    "installment_default",
    "risk_score",
}
ALLOWED_OPERATORS = {"eq", "neq", "gt", "gte", "lt", "lte", "in", "not_in"}
VALUE_ALIASES = {
    "risk_level": {"کم": "low", "متوسط": "medium", "بالا": "high", "low": "low", "medium": "medium", "high": "high"},
    "gender": {"مرد": "male", "زن": "female", "male": "male", "female": "female"},
    "occupation": {"کارمند": "employee", "خانه‌دار": "housewife", "خانه دار": "housewife", "مدیر": "manager", "بازنشسته": "retired", "بیکار": "unemployed", "دانشجو": "student", "شغل آزاد": "self_employed"},
    "employment_type": {"دولتی": "government", "خصوصی": "private", "آزاد": "freelance", "بدون اشتغال": "none"},
    "payment_history": {"عالی": "excellent", "خوب": "good", "متوسط": "fair", "ضعیف": "poor"},
    "spending_pattern": {"محافظه‌کارانه": "conservative", "متوسط": "moderate", "تهاجمی": "aggressive"},
}


SYSTEM_PROMPT = """You extract banking eligibility rules from Persian circular text.
Return JSON only. Never infer a condition that is not explicitly stated.
Every condition must include an exact source_quote copied from the input.
Allowed fields: {fields}
Allowed operators: {operators}
Output schema:
{{
  "description": "short English summary",
  "description_fa": "خلاصه فارسی",
  "conditions": [
    {{"field": "age", "operator": "gte", "value": 18, "source_quote": "exact quote"}}
  ],
  "warnings": ["ambiguities or missing information"]
}}
Use تومان for monetary values and convert written numbers to numeric values only when explicit.
"""


def configured_model() -> str:
    return os.getenv("LLM_MODEL") or os.getenv("OPENAI_MODEL") or DEFAULT_LLM_MODEL


def _content_text(content) -> str:
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "\n".join(
            block.get("text", "") if isinstance(block, dict) else str(block)
            for block in content
        )
    return str(content)


def _parse_json(text: str) -> dict:
    text = text.strip()
    fenced = re.search(r"```(?:json)?\s*(.*?)\s*```", text, re.S | re.I)
    if fenced:
        text = fenced.group(1)
    try:
        value = json.loads(text)
    except json.JSONDecodeError as exc:
        raise ValueError(f"LLM returned invalid JSON: {exc}") from exc
    if not isinstance(value, dict):
        raise ValueError("LLM output must be a JSON object")
    return value


def _normalized(value: str) -> str:
    return re.sub(r"\s+", " ", value or "").strip()


def _canonical_value(field: str, value):
    if isinstance(value, list):
        return [_canonical_value(field, item) for item in value]
    if isinstance(value, str):
        normalized = _normalized(value).lower()
        return VALUE_ALIASES.get(field, {}).get(normalized, value)
    return value


def validate_extraction(payload: dict, circular_text: str, product_id: str, circular_ref: str) -> dict:
    if product_id not in PRODUCTS:
        raise ValueError(f"unknown product_id: {product_id}")
    conditions = payload.get("conditions")
    if not isinstance(conditions, list) or not conditions:
        raise ValueError("no explicit eligibility conditions were extracted")

    source = _normalized(circular_text)
    validated = []
    for index, condition in enumerate(conditions):
        if not isinstance(condition, dict):
            raise ValueError(f"condition {index} is not an object")
        field = condition.get("field")
        operator = condition.get("operator")
        quote = _normalized(condition.get("source_quote", ""))
        value = _canonical_value(field, condition.get("value"))
        if field not in ALLOWED_FIELDS:
            raise ValueError(f"condition {index} has unsupported field: {field}")
        if operator not in ALLOWED_OPERATORS:
            raise ValueError(f"condition {index} has unsupported operator: {operator}")
        if not quote or quote not in source:
            raise ValueError(f"condition {index} source_quote is not present in the circular")
        if operator in {"in", "not_in"} and not isinstance(value, list):
            raise ValueError(f"condition {index} value must be a list for {operator}")
        if field in NUMERIC_FIELDS and operator not in {"in", "not_in"}:
            if isinstance(value, bool) or not isinstance(value, (int, float)):
                raise ValueError(f"condition {index} value must be numeric for {field}")
        validated.append({
            "field": field,
            "operator": operator,
            "value": value,
            "source_quote": quote,
        })

    return {
        "status": "draft",
        "requires_human_approval": True,
        "model": configured_model(),
        "circular_ref": circular_ref,
        "circular_ref_fa": f"بخشنامه شماره {circular_ref}",
        "product_id": product_id,
        "product_name_fa": PRODUCTS[product_id],
        "description": str(payload.get("description", "")).strip(),
        "description_fa": str(payload.get("description_fa", "")).strip(),
        "conditions": validated,
        "warnings": payload.get("warnings") if isinstance(payload.get("warnings"), list) else [],
    }


def extract_rule_draft(circular_text: str, product_id: str, circular_ref: str) -> dict:
    circular_text = circular_text.strip()
    if len(circular_text) < 20:
        raise ValueError("circular text is too short")
    if len(circular_text) > 50_000:
        raise ValueError("circular text exceeds 50000 characters")

    llm = ChatOpenAI(
        model=configured_model(),
        base_url=os.getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
        api_key=os.getenv("OPENAI_API_KEY", "not-set"),
        temperature=0,
        timeout=float(os.getenv("OPENAI_TIMEOUT_SECONDS", "30")),
        use_responses_api=False,
    )
    response = llm.invoke([
        SystemMessage(content=SYSTEM_PROMPT.format(
            fields=", ".join(sorted(ALLOWED_FIELDS)),
            operators=", ".join(sorted(ALLOWED_OPERATORS)),
        )),
        HumanMessage(content=(
            f"Product: {product_id} - {PRODUCTS.get(product_id, 'unknown')}\n"
            f"Circular reference: {circular_ref}\n"
            f"Circular text:\n{circular_text}"
        )),
    ])
    return validate_extraction(_parse_json(_content_text(response.content)), circular_text, product_id, circular_ref)


def main() -> None:
    parser = argparse.ArgumentParser(description="Extract a validated draft rule from a Persian circular")
    parser.add_argument("input", type=Path)
    parser.add_argument("--product-id", required=True, choices=sorted(PRODUCTS))
    parser.add_argument("--circular-ref", required=True)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()

    draft = extract_rule_draft(args.input.read_text(encoding="utf-8"), args.product_id, args.circular_ref)
    rendered = json.dumps(draft, ensure_ascii=False, indent=2)
    if args.output:
        args.output.write_text(rendered + "\n", encoding="utf-8")
    else:
        print(rendered)


if __name__ == "__main__":
    main()
