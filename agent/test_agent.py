"""
Tests for banking agent tools — runs against the Go backend.
Start backend first: cd backend && go run ./cmd/server
"""

import json
import os
import sys
import httpx

# Add parent to path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080")


class TestIdentityTool:
    def test_existing_customer(self):
        r = httpx.get(f"{BACKEND_URL}/api/identity", params={"national_id": "0012345678"})
        assert r.status_code == 200
        data = r.json()
        assert data["name"] == "فاطمه احمدی"
        assert data["age"] == 40
        assert data["gender"] == "female"
        assert data["occupation"] == "housewife"
        assert data["is_existing"] is True

    def test_non_existing_customer(self):
        r = httpx.get(f"{BACKEND_URL}/api/identity", params={"national_id": "9999999999"})
        assert r.status_code == 404

    def test_missing_param(self):
        r = httpx.get(f"{BACKEND_URL}/api/identity")
        assert r.status_code == 400

    def test_all_mock_customers(self):
        """Verify all 5 mock customers are accessible."""
        ids = ["0012345678", "0023456789", "0034567890", "0045678901", "0056789012"]
        for nid in ids:
            r = httpx.get(f"{BACKEND_URL}/api/identity", params={"national_id": nid})
            assert r.status_code == 200, f"Customer {nid} not found"


class TestFinancialTool:
    def test_existing(self):
        r = httpx.get(f"{BACKEND_URL}/api/financial", params={"customer_id": "C002"})
        assert r.status_code == 200
        data = r.json()
        assert data["monthly_income"] == 120_000_000
        assert data["payment_history"] == "excellent"

    def test_not_found(self):
        r = httpx.get(f"{BACKEND_URL}/api/financial", params={"customer_id": "CXXX"})
        assert r.status_code == 404


class TestRBCI:
    def test_existing_risk(self):
        r = httpx.get(f"{BACKEND_URL}/api/rbci", params={"customer_id": "C001"})
        assert r.status_code == 200
        data = r.json()
        assert data["risk_level"] == "medium"

    def test_cold_start(self):
        r = httpx.post(f"{BACKEND_URL}/api/rbci/cold-start", json={
            "name": "تست", "age": 35, "gender": "male",
            "occupation": "employee", "employment_type": "private",
            "approx_income": 30_000_000, "visit_purpose": "وام",
        })
        assert r.status_code == 200
        data = r.json()
        assert data["is_cold_start"] is True
        assert data["risk_level"] in ["low", "medium", "high"]

    def test_cold_start_bad_request(self):
        r = httpx.post(f"{BACKEND_URL}/api/rbci/cold-start", json={"name": "test"})
        assert r.status_code == 400


class TestProducts:
    def test_list(self):
        r = httpx.get(f"{BACKEND_URL}/api/products")
        assert r.status_code == 200
        products = r.json()
        assert len(products) == 10
        ids = {p["id"] for p in products}
        assert "P001" in ids  # personal loan
        assert "P003" in ids  # checkbook


class TestCirculars:
    def test_list_all(self):
        r = httpx.get(f"{BACKEND_URL}/api/circulars")
        assert r.status_code == 200
        rules = r.json()
        assert len(rules) == 10

    def test_by_product(self):
        r = httpx.get(f"{BACKEND_URL}/api/circulars/by-product", params={"product_id": "P001"})
        assert r.status_code == 200
        rules = r.json()
        assert len(rules) >= 1
        assert rules[0]["product_id"] == "P001"


class TestMatching:
    def test_housewife_scenario(self):
        """سناریو: خانم خانه‌دار ۴۰ ساله"""
        r = httpx.post(f"{BACKEND_URL}/api/match", json={
            "national_id": "0012345678",
            "include_default_warning": True,
        })
        assert r.status_code == 200
        data = r.json()
        assert data["customer_name"] == "فاطمه احمدی"
        assert data["is_existing"] is True

        # Should have some eligible and some ineligible
        eligible_ids = {p["product_id"] for p in data["eligible_products"]}
        ineligible_ids = {p["product_id"] for p in data["ineligible_products"]}

        # Housewife with 8M income should NOT get personal loan (requires 10M)
        assert "P001" in ineligible_ids, "Housewife should be ineligible for personal loan"

        # Should be eligible for deposits and basic services
        assert "P005" in eligible_ids, "Should be eligible for short-term deposit"

        # Gap analysis should exist for ineligible products
        for p in data["ineligible_products"]:
            assert len(p["gaps"]) > 0, f"Product {p['product_id']} should have gap analysis"

        # Default warning should be present
        assert data["default_warning"] is not None

    def test_manager_scenario(self):
        """سناریو: مدیر با درآمد بالا"""
        r = httpx.post(f"{BACKEND_URL}/api/match", json={
            "national_id": "0023456789",
            "include_default_warning": False,
        })
        assert r.status_code == 200
        data = r.json()
        eligible_ids = {p["product_id"] for p in data["eligible_products"]}

        assert "P003" in eligible_ids, "Manager should be eligible for checkbook"
        assert "P007" in eligible_ids, "Manager should be eligible for business loan"
        assert "P002" in eligible_ids, "Manager should be eligible for housing loan"

        # Should have personalized offers
        assert len(data["personalized_offers"]) > 0

    def test_employee_scenario(self):
        """سناریو: کارمند با اقساط معوق"""
        r = httpx.post(f"{BACKEND_URL}/api/match", json={
            "national_id": "0034567890",
            "include_default_warning": True,
        })
        assert r.status_code == 200
        data = r.json()
        ineligible_ids = {p["product_id"] for p in data["ineligible_products"]}
        eligible_ids = {p["product_id"] for p in data["eligible_products"]}

        # Employee with 2 defaults should NOT get checkbook (requires 0)
        assert "P003" in ineligible_ids, "Employee with defaults should be ineligible for checkbook"

        # But credit card allows up to 2 defaults
        assert "P004" in eligible_ids, "Employee should still be eligible for credit card"

    def test_non_customer_scenario(self):
        """سناریو: فرد غیرمشتری"""
        r = httpx.post(f"{BACKEND_URL}/api/match/cold-start", json={
            "name": "سارا نوروزی", "age": 30, "gender": "female",
            "occupation": "employee", "employment_type": "private",
            "approx_income": 20_000_000, "visit_purpose": "وام شخصی",
            "include_default_warning": True,
        })
        assert r.status_code == 200
        data = r.json()
        assert data["is_cold_start"] is True
        assert data["is_existing"] is False

        eligible_ids = {p["product_id"] for p in data["eligible_products"]}
        # Non-customer should be eligible for basic services
        assert "P008" in eligible_ids or "P009" in eligible_ids, \
            "Non-customer should be eligible for at least basic services"

    def test_non_customer_404(self):
        """Unknown national_id should return 404 on match"""
        r = httpx.post(f"{BACKEND_URL}/api/match", json={"national_id": "9999999999"})
        assert r.status_code == 404


class TestDefaultWarning:
    def test_included_when_requested(self):
        r = httpx.post(f"{BACKEND_URL}/api/match", json={
            "national_id": "0023456789",
            "include_default_warning": True,
        })
        data = r.json()
        assert data["default_warning"] is not None
        assert len(data["default_warning"]["consequences_fa"]) > 0

    def test_not_included_when_not_requested(self):
        r = httpx.post(f"{BACKEND_URL}/api/match", json={
            "national_id": "0023456789",
            "include_default_warning": False,
        })
        data = r.json()
        assert data.get("default_warning") is None
