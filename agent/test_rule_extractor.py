from types import SimpleNamespace

import pytest

import rule_extractor


CIRCULAR = "متقاضی باید حداقل ۱۸ سال سن و درآمد ماهانه حداقل ۱۰ میلیون تومان داشته باشد."


def test_validate_extraction_accepts_supported_evidenced_conditions(monkeypatch):
    monkeypatch.setenv("LLM_MODEL", "ag/gemini-3.6-flash-high")
    draft = rule_extractor.validate_extraction(
        {
            "description": "Personal loan minimums",
            "description_fa": "حداقل‌های وام شخصی",
            "conditions": [
                {"field": "age", "operator": "gte", "value": 18, "source_quote": "حداقل ۱۸ سال سن"},
                {
                    "field": "monthly_income",
                    "operator": "gte",
                    "value": 10_000_000,
                    "source_quote": "درآمد ماهانه حداقل ۱۰ میلیون تومان",
                },
            ],
            "warnings": [],
        },
        CIRCULAR,
        "P001",
        "BN-TEST/1",
    )

    assert draft["status"] == "draft"
    assert draft["requires_human_approval"] is True
    assert draft["model"] == "ag/gemini-3.6-flash-high"
    assert len(draft["conditions"]) == 2


@pytest.mark.parametrize(
    "condition,error",
    [
        (
            {"field": "credit_rating_secret", "operator": "gte", "value": 1, "source_quote": "حداقل ۱۸ سال سن"},
            "unsupported field",
        ),
        (
            {"field": "age", "operator": "approximately", "value": 18, "source_quote": "حداقل ۱۸ سال سن"},
            "unsupported operator",
        ),
        (
            {"field": "age", "operator": "gte", "value": 18, "source_quote": "عبارت ساختگی"},
            "source_quote is not present",
        ),
    ],
)
def test_validate_extraction_rejects_unsafe_conditions(condition, error):
    with pytest.raises(ValueError, match=error):
        rule_extractor.validate_extraction(
            {"conditions": [condition]},
            CIRCULAR,
            "P001",
            "BN-TEST/1",
        )


def test_extract_rule_draft_calls_configured_llm(monkeypatch):
    payload = {
        "description": "Age minimum",
        "description_fa": "حداقل سن",
        "conditions": [
            {"field": "age", "operator": "gte", "value": 18, "source_quote": "حداقل ۱۸ سال سن"}
        ],
        "warnings": [],
    }
    captured = {}

    class FakeLLM:
        def __init__(self, **kwargs):
            captured.update(kwargs)

        def invoke(self, messages):
            captured["messages"] = messages
            return SimpleNamespace(content=str_json)

    str_json = __import__("json").dumps(payload, ensure_ascii=False)
    monkeypatch.setattr(rule_extractor, "ChatOpenAI", FakeLLM)
    monkeypatch.setenv("LLM_MODEL", "ag/gemini-3.6-flash-high")

    draft = rule_extractor.extract_rule_draft(CIRCULAR, "P001", "BN-TEST/1")

    assert captured["model"] == "ag/gemini-3.6-flash-high"
    assert captured["temperature"] == 0
    assert draft["conditions"][0]["field"] == "age"


def test_validate_extraction_normalizes_persian_enum_values():
    draft = rule_extractor.validate_extraction(
        {
            "conditions": [
                {
                    "field": "risk_level",
                    "operator": "in",
                    "value": ["کم", "متوسط"],
                    "source_quote": "ریسک کم یا متوسط",
                }
            ]
        },
        "سطح ریسک باید ریسک کم یا متوسط باشد.",
        "P001",
        "BN-TEST/2",
    )

    assert draft["conditions"][0]["value"] == ["low", "medium"]
