// تست‌های HTTP لایه handlers
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/banking-assistant/backend/internal/models"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	HealthHandler(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestIdentityHandler_Existing(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/identity?national_id=0012345678", nil)
	w := httptest.NewRecorder()
	IdentityHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var profile models.IdentityProfile
	json.NewDecoder(w.Body).Decode(&profile)
	if profile.Name != "فاطمه احمدی" {
		t.Errorf("expected فاطمه احمدی, got %s", profile.Name)
	}
	if profile.Age != 40 {
		t.Errorf("expected age 40, got %d", profile.Age)
	}
}

func TestIdentityHandler_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/identity?national_id=9999999999", nil)
	w := httptest.NewRecorder()
	IdentityHandler(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestIdentityHandler_InvalidFormats(t *testing.T) {
	cases := []struct {
		nid  string
		code int
	}{
		{"", 400},
		{"123", 400},
		{"abcdefghij", 400},
		{"0000000000", 400},
		{"12345678901", 400}, // 11 digits
	}
	for _, tc := range cases {
		req := httptest.NewRequest("GET", "/api/identity?national_id="+tc.nid, nil)
		w := httptest.NewRecorder()
		IdentityHandler(w, req)
		if w.Code != tc.code {
			t.Errorf("nid=%q: expected %d, got %d", tc.nid, tc.code, w.Code)
		}
	}
}

func TestFinancialHandler_Existing(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/financial?customer_id=C002", nil)
	w := httptest.NewRecorder()
	FinancialHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var profile models.FinancialProfile
	json.NewDecoder(w.Body).Decode(&profile)
	if profile.MonthlyIncome != 120_000_000 {
		t.Errorf("expected income 120M, got %v", profile.MonthlyIncome)
	}
}

func TestFinancialHandler_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/financial?customer_id=CXXX", nil)
	w := httptest.NewRecorder()
	FinancialHandler(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRBCIHandler_Existing(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/rbci?customer_id=C001", nil)
	w := httptest.NewRecorder()
	RBCIHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var risk models.RiskAssessment
	json.NewDecoder(w.Body).Decode(&risk)
	if risk.RiskLevel != "medium" {
		t.Errorf("expected medium risk, got %s", risk.RiskLevel)
	}
}

func TestColdStartHandler(t *testing.T) {
	body, _ := json.Marshal(models.ColdStartRequest{
		Name: "تست", Age: 35, Gender: "male",
		Occupation: "employee", EmploymentType: "private",
		ApproxIncome: 30_000_000,
	})
	req := httptest.NewRequest("POST", "/api/rbci/cold-start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	ColdStartHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var risk models.RiskAssessment
	json.NewDecoder(w.Body).Decode(&risk)
	if !risk.IsColdStart {
		t.Error("expected cold-start flag")
	}
}

func TestColdStartHandler_PersianOccupation(t *testing.T) {
	body, _ := json.Marshal(models.ColdStartRequest{
		Name: "علی", Age: 40, Gender: "male",
		Occupation: "مدیر", EmploymentType: "private",
		ApproxIncome: 80_000_000,
	})
	req := httptest.NewRequest("POST", "/api/rbci/cold-start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	ColdStartHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 for Persian occupation, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestColdStartHandler_BadRequest(t *testing.T) {
	cases := []map[string]interface{}{
		{"name": "test"},                         // missing age/occupation
		{"name": "t", "age": 10, "occupation": "employee"}, // age too low
		{"name": "t", "age": 30, "occupation": "employee", "gender": "other"},
		{"name": "t", "age": 30, "occupation": "employee", "approx_income": -1},
		{"age": 30, "occupation": "employee"}, // missing name
	}
	for i, c := range cases {
		body, _ := json.Marshal(c)
		req := httptest.NewRequest("POST", "/api/rbci/cold-start", bytes.NewReader(body))
		w := httptest.NewRecorder()
		ColdStartHandler(w, req)
		if w.Code != 400 {
			t.Errorf("case %d: expected 400, got %d body=%s", i, w.Code, w.Body.String())
		}
	}
}

func TestProductsHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/products", nil)
	w := httptest.NewRecorder()
	ProductsHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var products []models.Product
	json.NewDecoder(w.Body).Decode(&products)
	if len(products) < 5 {
		t.Errorf("expected at least 5 products, got %d", len(products))
	}
}

func TestCircularsHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/circulars", nil)
	w := httptest.NewRecorder()
	CircularsHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var rules []models.CircularRule
	json.NewDecoder(w.Body).Decode(&rules)
	if len(rules) < 5 {
		t.Errorf("expected at least 5 rules, got %d", len(rules))
	}
}

func TestMatchHandler_ExistingCustomer(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"national_id":             "0023456789",
		"include_default_warning": true,
		"visit_purpose":           "دسته‌چک",
	})
	req := httptest.NewRequest("POST", "/api/match", bytes.NewReader(body))
	w := httptest.NewRecorder()
	MatchHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp models.MatchResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.CustomerName != "علی رضایی" {
		t.Errorf("expected علی رضایی, got %s", resp.CustomerName)
	}
	if len(resp.EligibleProducts) == 0 {
		t.Error("expected at least some eligible products")
	}
	if resp.DefaultWarning == nil {
		t.Error("expected default warning when requested")
	}
	// Visit purpose should boost checkbook in offers
	if len(resp.PersonalizedOffers) == 0 {
		t.Error("expected personalized offers")
	}
}

func TestMatchHandler_NotFound(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"national_id": "9999999999"})
	req := httptest.NewRequest("POST", "/api/match", bytes.NewReader(body))
	w := httptest.NewRecorder()
	MatchHandler(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404 for unknown customer, got %d", w.Code)
	}
}

func TestMatchHandler_InvalidNationalID(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"national_id": "abc"})
	req := httptest.NewRequest("POST", "/api/match", bytes.NewReader(body))
	w := httptest.NewRecorder()
	MatchHandler(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid nid, got %d", w.Code)
	}
}

func TestMatchColdStartHandler(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"name": "سارا نوروزی", "age": 30, "gender": "female",
		"occupation": "employee", "employment_type": "private",
		"approx_income": 20_000_000, "visit_purpose": "وام",
		"include_default_warning": true,
	})
	req := httptest.NewRequest("POST", "/api/match/cold-start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	MatchColdStartHandler(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp models.MatchResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.IsColdStart {
		t.Error("expected cold-start flag")
	}
	if resp.IsExisting {
		t.Error("expected is_existing=false")
	}
	if len(resp.NotesFa) == 0 {
		t.Error("expected cold-start notes_fa")
	}
	// Eligible products for non-customer must be conditional
	for _, p := range resp.EligibleProducts {
		if !p.IsConditional {
			t.Errorf("product %s should be conditional for non-customer", p.ProductID)
		}
		if len(p.ConditionsFa) == 0 {
			t.Errorf("product %s missing conditions_fa", p.ProductID)
		}
	}
}
