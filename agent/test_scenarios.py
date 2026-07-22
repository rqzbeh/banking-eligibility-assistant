"""
End-to-end scenario tests — validates all 5 challenge scenarios.
Starts Go backend automatically.
"""

import json
import os
import subprocess
import time
import signal
import pytest
import httpx

BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080")


@pytest.fixture(scope="session", autouse=True)
def backend_server():
    backend_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "backend")
    proc = subprocess.Popen(
        ["go", "run", "./cmd/server"],
        cwd=backend_dir,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        preexec_fn=os.setsid,
    )
    for _ in range(30):
        try:
            r = httpx.get(f"{BACKEND_URL}/api/health", timeout=1)
            if r.status_code == 200:
                break
        except Exception:
            time.sleep(0.5)
    else:
        proc.kill()
        raise RuntimeError("Backend failed to start")
    yield proc
    os.killpg(os.getpgid(proc.pid), signal.SIGTERM)
    proc.wait(timeout=5)


client = httpx.Client(base_url=BACKEND_URL, timeout=10)


class TestScenario1_Housewife:
    """سناریو ۱: خانم خانه‌دار ۴۰ ساله — فاطمه احمدی"""

    def test_identity_found(self):
        r = client.get("/api/identity", params={"national_id": "0012345678"})
        assert r.status_code == 200
        d = r.json()
        assert d["age"] == 40
        assert d["gender"] == "female"
        assert d["occupation"] == "housewife"

    def test_full_match(self):
        r = client.post("/api/match", json={"national_id": "0012345678", "include_default_warning": True})
        assert r.status_code == 200
        d = r.json()

        elig_ids = {p["product_id"] for p in d["eligible_products"]}
        inelig_ids = {p["product_id"] for p in d["ineligible_products"]}

        # Income 8M < 10M → no personal loan
        assert "P001" in inelig_ids

        # Turnover 30M < 100M → no checkbook
        assert "P003" in inelig_ids

        # Income 8M < 15M → no credit card
        assert "P004" in inelig_ids

        # Deposits and basic services should be eligible
        assert "P005" in elig_ids  # short-term deposit
        assert "P008" in elig_ids  # online banking
        assert "P009" in elig_ids  # SMS banking

    def test_gap_analysis_quality(self):
        r = client.post("/api/match", json={"national_id": "0012345678"})
        d = r.json()

        for p in d["ineligible_products"]:
            assert len(p["gaps"]) > 0, f"No gap analysis for {p['product_name']}"
            assert len(p["circular_refs"]) > 0, f"No circular refs for {p['product_name']}"
            for g in p["gaps"]:
                assert g["description_fa"], f"Missing Farsi description in gap for {p['product_name']}"
                assert g["current_value"], f"Missing current value in gap"
                assert g["required_value"], f"Missing required value in gap"

    def test_personalized_offers(self):
        r = client.post("/api/match", json={"national_id": "0012345678"})
        d = r.json()
        assert len(d["personalized_offers"]) > 0, "Should have at least one personalized offer"


class TestScenario2_HighIncomeManager:
    """سناریو ۲: مدیر با درآمد بالا — علی رضایی"""

    def test_identity(self):
        r = client.get("/api/identity", params={"national_id": "0023456789"})
        d = r.json()
        assert d["occupation"] == "manager"

    def test_checkbook_eligible(self):
        r = client.post("/api/match", json={"national_id": "0023456789"})
        d = r.json()
        elig = {p["product_id"]: p for p in d["eligible_products"]}
        assert "P003" in elig, "Manager should be eligible for checkbook"
        cb = elig["P003"]
        assert cb.get("obligations_fa"), "Checkbook must show obligations"
        assert cb.get("credit_limit_fa"), "Checkbook must show credit limit"

    def test_all_loans_eligible(self):
        r = client.post("/api/match", json={"national_id": "0023456789"})
        d = r.json()
        elig_ids = {p["product_id"] for p in d["eligible_products"]}
        assert "P001" in elig_ids, "Should be eligible for personal loan"
        assert "P002" in elig_ids, "Should be eligible for housing loan"
        assert "P007" in elig_ids, "Should be eligible for business loan"
        assert "P004" in elig_ids, "Should be eligible for credit card"

    def test_risk_is_low(self):
        r = client.get("/api/rbci", params={"customer_id": "C002"})
        d = r.json()
        assert d["risk_level"] == "low"

    def test_offers_ranked(self):
        r = client.post("/api/match", json={"national_id": "0023456789"})
        d = r.json()
        offers = d["personalized_offers"]
        assert len(offers) >= 2
        # Scores should be descending
        for i in range(len(offers) - 1):
            assert offers[i]["score"] >= offers[i + 1]["score"], "Offers not sorted by score"


class TestScenario3_EmployeeWithDefaults:
    """سناریو ۳: کارمند — محمد حسینی (اقساط معوق)"""

    def test_defaults_detected(self):
        r = client.get("/api/financial", params={"customer_id": "C003"})
        d = r.json()
        assert d["installment_default"] == 2

    def test_checkbook_ineligible_with_gap(self):
        r = client.post("/api/match", json={"national_id": "0034567890"})
        d = r.json()
        inelig = {p["product_id"]: p for p in d["ineligible_products"]}
        assert "P003" in inelig, "Should be ineligible for checkbook"

        # Gap should mention installment_default
        check_gaps = inelig["P003"]["gaps"]
        default_gap = [g for g in check_gaps if g["field"] == "installment_default"]
        assert len(default_gap) > 0, "Gap should mention installment defaults"

    def test_credit_card_eligible(self):
        """Credit card allows up to 2 defaults"""
        r = client.post("/api/match", json={"national_id": "0034567890"})
        d = r.json()
        elig_ids = {p["product_id"] for p in d["eligible_products"]}
        assert "P004" in elig_ids, "Should be eligible for credit card (2 defaults <= 2 max)"

    def test_path_to_checkbook(self):
        """Verify gap analysis shows what needs to change for checkbook eligibility"""
        r = client.post("/api/match", json={"national_id": "0034567890"})
        d = r.json()
        inelig = {p["product_id"]: p for p in d["ineligible_products"]}
        gaps = inelig["P003"]["gaps"]
        # Should have actionable gap descriptions in Farsi
        for g in gaps:
            assert "فعلی" in g["description_fa"] or "باید" in g["description_fa"], \
                f"Gap description should be actionable: {g['description_fa']}"
            assert g.get("advice_fa"), f"Gap must have advice_fa: {g}"
        # Product-level actionable advice for improving standing
        assert inelig["P003"].get("advice_fa"), "Must have advice_fa steps toward checkbook eligibility"


class TestScenario4_NonCustomer:
    """سناریو ۴: فرد غیرمشتری"""

    def test_404_on_identity(self):
        r = client.get("/api/identity", params={"national_id": "1111111111"})
        assert r.status_code == 404

    def test_404_on_match(self):
        r = client.post("/api/match", json={"national_id": "1111111111"})
        assert r.status_code == 404

    def test_cold_start_match(self):
        r = client.post("/api/match/cold-start", json={
            "name": "سارا نوروزی", "age": 30, "gender": "female",
            "occupation": "employee", "employment_type": "private",
            "approx_income": 20_000_000, "visit_purpose": "وام شخصی",
        })
        assert r.status_code == 200
        d = r.json()
        assert d["is_cold_start"] is True
        assert d["is_existing"] is False

    def test_basic_services_eligible(self):
        r = client.post("/api/match/cold-start", json={
            "name": "سارا", "age": 25, "gender": "female",
            "occupation": "employee", "employment_type": "private",
            "approx_income": 15_000_000, "visit_purpose": "بانکداری",
        })
        d = r.json()
        elig_ids = {p["product_id"] for p in d["eligible_products"]}
        assert "P008" in elig_ids, "Non-customer should be eligible for online banking"
        assert "P009" in elig_ids, "Non-customer should be eligible for SMS banking"

    def test_conditional_loans(self):
        """Non-customer with no turnover should be ineligible for turnover-dependent products"""
        r = client.post("/api/match/cold-start", json={
            "name": "سارا", "age": 25, "gender": "female",
            "occupation": "employee", "employment_type": "private",
            "approx_income": 50_000_000, "visit_purpose": "وام",
        })
        d = r.json()
        inelig_ids = {p["product_id"] for p in d["ineligible_products"]}
        # Checkbook requires 100M 3M turnover — non-customer has 0
        assert "P003" in inelig_ids, "Non-customer should be ineligible for checkbook"

    def test_cold_start_risk_levels(self):
        """Different profiles should get different risk levels"""
        # High-income manager → low risk
        r1 = client.post("/api/rbci/cold-start", json={
            "name": "t", "age": 40, "gender": "male",
            "occupation": "manager", "employment_type": "government",
            "approx_income": 80_000_000, "visit_purpose": "test",
        })
        # Unemployed youth → high risk
        r2 = client.post("/api/rbci/cold-start", json={
            "name": "t", "age": 19, "gender": "male",
            "occupation": "unemployed", "employment_type": "none",
            "approx_income": 0, "visit_purpose": "test",
        })
        d1, d2 = r1.json(), r2.json()
        assert d1["risk_score"] < d2["risk_score"], \
            f"Manager should have lower risk score ({d1['risk_score']}) than unemployed ({d2['risk_score']})"


class TestScenario5_DefaultWarning:
    """سناریو ۵: پیامدهای عدم پرداخت تعهدات"""

    def test_low_risk_consequences(self):
        r = client.post("/api/match", json={
            "national_id": "0023456789",  # low risk customer
            "include_default_warning": True,
        })
        d = r.json()
        w = d["default_warning"]
        assert w["current_risk_level"] == "low"
        assert w["potential_risk_level"] == "medium"
        assert len(w["consequences_fa"]) >= 3

    def test_medium_risk_consequences(self):
        r = client.post("/api/match", json={
            "national_id": "0012345678",  # medium risk customer
            "include_default_warning": True,
        })
        d = r.json()
        w = d["default_warning"]
        assert w["current_risk_level"] == "medium"
        assert w["potential_risk_level"] == "high"
        assert any("حقوقی" in c for c in w["consequences_fa"]), \
            "Medium→high should mention legal consequences"

    def test_no_warning_when_not_requested(self):
        r = client.post("/api/match", json={
            "national_id": "0023456789",
            "include_default_warning": False,
        })
        d = r.json()
        assert d.get("default_warning") is None


class TestEdgeCases:
    """تست موارد مرزی"""

    def test_empty_national_id(self):
        r = client.get("/api/identity", params={"national_id": ""})
        assert r.status_code == 400

    def test_cold_start_missing_fields(self):
        r = client.post("/api/rbci/cold-start", json={"name": "test"})
        assert r.status_code == 400

    def test_all_products_have_rules(self):
        """Every product that's not a basic service should have circular rules"""
        products = client.get("/api/products").json()
        circulars = client.get("/api/circulars").json()
        products_with_rules = {c["product_id"] for c in circulars}
        for p in products:
            assert p["id"] in products_with_rules, f"Product {p['id']} ({p['name_fa']}) has no circular rules"

    def test_consistency_circular_refs(self):
        """All circular refs should point to existing products"""
        products = client.get("/api/products").json()
        product_ids = {p["id"] for p in products}
        circulars = client.get("/api/circulars").json()
        for c in circulars:
            assert c["product_id"] in product_ids, f"Circular {c['id']} references unknown product {c['product_id']}"

    def test_match_response_structure(self):
        """Verify response has all required fields"""
        r = client.post("/api/match", json={"national_id": "0023456789", "include_default_warning": True})
        d = r.json()
        required_fields = ["customer_id", "national_id", "customer_name", "is_existing",
                          "risk_level", "risk_score", "eligible_products", "ineligible_products",
                          "personalized_offers", "default_warning"]
        for f in required_fields:
            assert f in d, f"Missing field: {f}"

    def test_invalid_national_id_formats(self):
        for nid in ["123", "abcdefghij", "0000000000", "12345678901"]:
            r = client.get("/api/identity", params={"national_id": nid})
            assert r.status_code == 400, f"nid={nid} should be 400"

    def test_housewife_alternatives_for_loan(self):
        """PDF: مسیر جایگزین ضامن/سپرده/گردش برای خانه‌دار"""
        r = client.post("/api/match", json={"national_id": "0012345678"})
        d = r.json()
        loan = next(p for p in d["ineligible_products"] if p["product_id"] == "P001")
        alts = " ".join(loan.get("alternatives_fa") or [])
        assert alts, "loan must include alternatives_fa"
        assert ("ضامن" in alts) or ("سپرده" in alts) or ("گردش" in alts)

    def test_cold_start_conditional_offers(self):
        r = client.post("/api/match/cold-start", json={
            "name": "سارا", "age": 30, "gender": "female",
            "occupation": "employee", "employment_type": "private",
            "approx_income": 25_000_000, "visit_purpose": "وام شخصی",
        })
        d = r.json()
        assert d.get("notes_fa"), "cold-start must return notes_fa"
        assert d["eligible_products"], "should have some eligible products"
        for p in d["eligible_products"]:
            assert p.get("is_conditional") is True, f"{p['product_id']} should be conditional"
            assert p.get("conditions_fa"), f"{p['product_id']} missing conditions_fa"

    def test_visit_purpose_boosts_checkbook(self):
        r1 = client.post("/api/match", json={"national_id": "0023456789"})
        r2 = client.post("/api/match", json={
            "national_id": "0023456789", "visit_purpose": "دسته‌چک",
        })
        d1, d2 = r1.json(), r2.json()
        s1 = next(p["score"] for p in d1["eligible_products"] if p["product_id"] == "P003")
        s2 = next(p["score"] for p in d2["eligible_products"] if p["product_id"] == "P003")
        assert s2 > s1, f"visit purpose should boost checkbook: {s1} -> {s2}"

    def test_persian_occupation_cold_start(self):
        r = client.post("/api/rbci/cold-start", json={
            "name": "علی", "age": 40, "gender": "male",
            "occupation": "مدیر", "employment_type": "private",
            "approx_income": 80_000_000,
        })
        assert r.status_code == 200
        assert r.json()["is_cold_start"] is True

    def test_gap_advice_on_ineligible(self):
        r = client.post("/api/match", json={"national_id": "0034567890"})
        d = r.json()
        for p in d["ineligible_products"]:
            if p["product_id"] == "P003":
                assert p.get("advice_fa"), "ineligible checkbook needs advice_fa"
                for g in p.get("gaps") or []:
                    if g["field"] == "installment_default":
                        assert g.get("advice_fa")
