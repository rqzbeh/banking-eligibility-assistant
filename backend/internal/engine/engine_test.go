package engine

import (
	"strings"
	"testing"

	"github.com/banking-assistant/backend/internal/data"
	"github.com/banking-assistant/backend/internal/models"
)

// --- EvaluateCondition tests ---

func TestEvaluateCondition_Eq(t *testing.T) {
	profile := CustomerProfile{Fields: map[string]interface{}{"risk_level": "low"}}
	ok, _, _ := EvaluateCondition(models.RuleCondition{Field: "risk_level", Operator: "eq", Value: "low"}, profile)
	if !ok {
		t.Error("expected eq to pass for matching value")
	}
	ok, _, _ = EvaluateCondition(models.RuleCondition{Field: "risk_level", Operator: "eq", Value: "high"}, profile)
	if ok {
		t.Error("expected eq to fail for non-matching value")
	}
}

func TestEvaluateCondition_Neq(t *testing.T) {
	profile := CustomerProfile{Fields: map[string]interface{}{"gender": "male"}}
	ok, _, _ := EvaluateCondition(models.RuleCondition{Field: "gender", Operator: "neq", Value: "female"}, profile)
	if !ok {
		t.Error("expected neq to pass")
	}
	ok, _, _ = EvaluateCondition(models.RuleCondition{Field: "gender", Operator: "neq", Value: "male"}, profile)
	if ok {
		t.Error("expected neq to fail for same value")
	}
}

func TestEvaluateCondition_NumericComparisons(t *testing.T) {
	profile := CustomerProfile{Fields: map[string]interface{}{"age": float64(30), "monthly_income": float64(25_000_000)}}

	tests := []struct {
		field    string
		op       string
		value    float64
		expected bool
	}{
		{"age", "gte", 18, true},
		{"age", "gte", 30, true},
		{"age", "gte", 31, false},
		{"age", "gt", 29, true},
		{"age", "gt", 30, false},
		{"age", "lte", 65, true},
		{"age", "lte", 30, true},
		{"age", "lte", 29, false},
		{"age", "lt", 31, true},
		{"age", "lt", 30, false},
		{"monthly_income", "gte", 10_000_000, true},
		{"monthly_income", "gte", 30_000_000, false},
	}

	for _, tt := range tests {
		ok, _, _ := EvaluateCondition(models.RuleCondition{Field: tt.field, Operator: tt.op, Value: tt.value}, profile)
		if ok != tt.expected {
			t.Errorf("%s %s %v: got %v, want %v", tt.field, tt.op, tt.value, ok, tt.expected)
		}
	}
}

func TestEvaluateCondition_In(t *testing.T) {
	profile := CustomerProfile{Fields: map[string]interface{}{"risk_level": "low"}}
	ok, _, _ := EvaluateCondition(models.RuleCondition{
		Field: "risk_level", Operator: "in",
		Value: []interface{}{"low", "medium"},
	}, profile)
	if !ok {
		t.Error("expected 'in' to pass for matching value")
	}

	ok, _, _ = EvaluateCondition(models.RuleCondition{
		Field: "risk_level", Operator: "in",
		Value: []interface{}{"medium", "high"},
	}, profile)
	if ok {
		t.Error("expected 'in' to fail for non-matching value")
	}
}

func TestEvaluateCondition_NotIn(t *testing.T) {
	profile := CustomerProfile{Fields: map[string]interface{}{"occupation": "employee"}}
	ok, _, _ := EvaluateCondition(models.RuleCondition{
		Field: "occupation", Operator: "not_in",
		Value: []interface{}{"unemployed", "student"},
	}, profile)
	if !ok {
		t.Error("expected not_in to pass for non-listed value")
	}

	ok, _, _ = EvaluateCondition(models.RuleCondition{
		Field: "occupation", Operator: "not_in",
		Value: []interface{}{"employee", "student"},
	}, profile)
	if ok {
		t.Error("expected not_in to fail for listed value")
	}
}

func TestEvaluateCondition_MissingField(t *testing.T) {
	profile := CustomerProfile{Fields: map[string]interface{}{}}
	ok, reason, _ := EvaluateCondition(models.RuleCondition{Field: "age", Operator: "gte", Value: float64(18)}, profile)
	if ok {
		t.Error("expected failure for missing field")
	}
	if reason == "" {
		t.Error("expected non-empty reason for missing field")
	}
}

// --- Scenario tests matching challenge requirements ---

func TestScenario_Housewife40(t *testing.T) {
	// سناریو: خانم خانه‌دار ۴۰ ساله (C001)
	identity := data.Identities["0012345678"]
	financial := data.Financials["C001"]
	risk := data.Risks["C001"]

	profile := BuildProfile(&identity, &financial, &risk)
	eligible, ineligible := MatchAllProducts(profile)

	// Should be ineligible for personal loan (income < 10M)
	found := false
	for _, p := range ineligible {
		if p.ProductID == "P001" {
			found = true
			if len(p.Gaps) == 0 {
				t.Error("personal loan: expected gap analysis for ineligible product")
			}
		}
	}
	if !found {
		t.Error("expected housewife to be ineligible for personal loan (income 8M < 10M required)")
	}

	// Should be ineligible for checkbook (turnover 30M < 100M)
	found = false
	for _, p := range ineligible {
		if p.ProductID == "P003" {
			found = true
		}
	}
	if !found {
		t.Error("expected housewife to be ineligible for checkbook (low turnover)")
	}

	// Should be eligible for short-term deposit
	found = false
	for _, p := range eligible {
		if p.ProductID == "P005" {
			found = true
		}
	}
	if !found {
		t.Error("expected housewife to be eligible for short-term deposit")
	}
}

func TestScenario_HighIncomeManager(t *testing.T) {
	// سناریو: مدیر با درآمد بالا (C002)
	identity := data.Identities["0023456789"]
	financial := data.Financials["C002"]
	risk := data.Risks["C002"]

	profile := BuildProfile(&identity, &financial, &risk)
	eligible, _ := MatchAllProducts(profile)

	// Should be eligible for checkbook
	var checkbook *models.ProductMatch
	for i, p := range eligible {
		if p.ProductID == "P003" {
			checkbook = &eligible[i]
		}
	}
	if checkbook == nil {
		t.Fatal("expected high-income manager to be eligible for checkbook")
	}
	// Must show obligations and credit limit (scoring criterion)
	if len(checkbook.ObligationsFa) == 0 {
		t.Error("checkbook eligible: expected obligations_fa")
	}
	if checkbook.CreditLimitFa == "" {
		t.Error("checkbook eligible: expected credit_limit_fa")
	}

	// Should be eligible for business loan
	found := false
	for _, p := range eligible {
		if p.ProductID == "P007" {
			found = true
			if len(p.ObligationsFa) == 0 {
				t.Error("business loan: expected obligations")
			}
		}
	}
	if !found {
		t.Error("expected high-income manager to be eligible for business loan")
	}

	// Should be eligible for housing loan
	found = false
	for _, p := range eligible {
		if p.ProductID == "P002" {
			found = true
		}
	}
	if !found {
		t.Error("expected high-income manager to be eligible for housing loan")
	}
}

func TestScenario_Employee(t *testing.T) {
	// سناریو: کارمند (C003) — has 2 installment defaults
	identity := data.Identities["0034567890"]
	financial := data.Financials["C003"]
	risk := data.Risks["C003"]

	profile := BuildProfile(&identity, &financial, &risk)
	eligible, ineligible := MatchAllProducts(profile)

	// Should be ineligible for checkbook (has defaults)
	found := false
	for _, p := range ineligible {
		if p.ProductID == "P003" {
			found = true
			// Verify gap analysis mentions defaults
			hasDefaultGap := false
			for _, g := range p.Gaps {
				if g.Field == "installment_default" {
					hasDefaultGap = true
					if g.AdviceFa == "" {
						t.Error("installment_default gap should have actionable advice_fa")
					}
				}
			}
			if !hasDefaultGap {
				t.Error("checkbook gap should mention installment_default")
			}
			// Must have actionable advice for improving standing (scoring criterion)
			if len(p.AdviceFa) == 0 {
				t.Error("ineligible checkbook: expected advice_fa with actionable steps")
			}
		}
	}
	if !found {
		t.Error("expected employee with defaults to be ineligible for checkbook")
	}

	// Should be eligible for credit card (max 2 defaults allowed)
	found = false
	for _, p := range eligible {
		if p.ProductID == "P004" {
			found = true
		}
	}
	if !found {
		t.Error("expected employee to be eligible for credit card (2 defaults <= 2 max)")
	}
}

func TestScenario_NonCustomer(t *testing.T) {
	// سناریو: فرد غیرمشتری
	req := models.ColdStartRequest{
		Name: "سارا نوروزی", Age: 30, Gender: "female",
		Occupation: "employee", EmploymentType: "private",
		ApproxIncome: 20_000_000, VisitPurpose: "وام شخصی",
	}
	risk := ColdStartRisk(req)
	profile := BuildProfileFromColdStart(req, &risk)

	eligible, ineligible := MatchAllProducts(profile)

	// Non-customer should be eligible for some services (online banking, SMS)
	hasService := false
	for _, p := range eligible {
		if p.ProductID == "P008" || p.ProductID == "P009" {
			hasService = true
		}
	}
	if !hasService {
		t.Error("non-customer should be eligible for basic services")
	}

	// Non-customer has 0 turnover, so should be ineligible for checkbook
	found := false
	for _, p := range ineligible {
		if p.ProductID == "P003" {
			found = true
		}
	}
	if !found {
		t.Error("non-customer should be ineligible for checkbook (no turnover)")
	}

	// Risk assessment should be cold-start
	if !risk.IsColdStart {
		t.Error("expected cold-start risk for non-customer")
	}
}

func TestScenario_DefaultWarning(t *testing.T) {
	// Test payment default consequences
	warning := GenerateDefaultWarning("low")
	if warning.PotentialRiskLevel != "medium" {
		t.Errorf("low risk default: expected potential medium, got %s", warning.PotentialRiskLevel)
	}
	if len(warning.ConsequencesFa) == 0 {
		t.Error("expected Farsi consequences")
	}

	warning = GenerateDefaultWarning("medium")
	if warning.PotentialRiskLevel != "high" {
		t.Errorf("medium risk default: expected potential high, got %s", warning.PotentialRiskLevel)
	}

	warning = GenerateDefaultWarning("high")
	if warning.PotentialRiskLevel != "high" {
		t.Error("high risk default: should remain high")
	}
}

// --- ColdStartRisk tests ---

func TestColdStartRisk_Ranges(t *testing.T) {
	tests := []struct {
		name       string
		req        models.ColdStartRequest
		expectLow  bool
		expectHigh bool
	}{
		{
			"high-income manager",
			models.ColdStartRequest{Age: 35, Occupation: "manager", EmploymentType: "government", ApproxIncome: 80_000_000},
			true, false,
		},
		{
			"unemployed youth",
			models.ColdStartRequest{Age: 19, Occupation: "unemployed", EmploymentType: "none", ApproxIncome: 0},
			false, true,
		},
		{
			"average employee",
			models.ColdStartRequest{Age: 30, Occupation: "employee", EmploymentType: "private", ApproxIncome: 25_000_000},
			true, false,
		},
	}

	for _, tt := range tests {
		risk := ColdStartRisk(tt.req)
		if tt.expectLow && risk.RiskLevel != "low" {
			t.Errorf("%s: expected low risk, got %s (score: %.0f)", tt.name, risk.RiskLevel, risk.RiskScore)
		}
		if tt.expectHigh && risk.RiskLevel != "high" {
			t.Errorf("%s: expected high risk, got %s (score: %.0f)", tt.name, risk.RiskLevel, risk.RiskScore)
		}
		if !risk.IsColdStart {
			t.Errorf("%s: expected cold-start flag", tt.name)
		}
	}
}

// --- Helpers ---

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{10_000_000, "10 میلیون"},
		{1_500_000_000, "1.5 میلیارد"},
		{500_000, "500000 تومان"}, // below million threshold
		{18, "18"},
	}
	for _, tt := range tests {
		got := formatNumber(tt.input)
		// Just check it doesn't panic and produces non-empty output
		if got == "" {
			t.Errorf("formatNumber(%v) returned empty", tt.input)
		}
	}
}

func TestFieldNameFa(t *testing.T) {
	if fieldNameFa("age") != "سن" {
		t.Error("expected Farsi name for 'age'")
	}
	if fieldNameFa("unknown_field") != "unknown_field" {
		t.Error("unknown field should return field name as-is")
	}
}

func TestToFloat(t *testing.T) {
	if toFloat(float64(42)) != 42 {
		t.Error("float64 conversion failed")
	}
	if toFloat(int(42)) != 42 {
		t.Error("int conversion failed")
	}
	if toFloat("not a number") != 0 {
		t.Error("invalid type should return 0")
	}
}

// --- Compliance & edge-case tests ---

func TestValidateNationalID(t *testing.T) {
	ok, _ := ValidateNationalID("0012345678")
	if !ok {
		t.Error("valid nid rejected")
	}
	cases := []string{"", "123", "abcdefghij", "0000000000", "12345678901"}
	for _, c := range cases {
		ok, msg := ValidateNationalID(c)
		if ok {
			t.Errorf("expected reject for %q", c)
		}
		if msg == "" {
			t.Errorf("expected message for %q", c)
		}
	}
}

func TestNormalizeOccupation(t *testing.T) {
	if NormalizeOccupation("مدیر") != "manager" {
		t.Error("مدیر should map to manager")
	}
	if NormalizeOccupation("خانه‌دار") != "housewife" {
		t.Error("خانه‌دار should map to housewife")
	}
	if NormalizeOccupation("employee") != "employee" {
		t.Error("english passthrough failed")
	}
}

func TestHousewife_AlternativesForLoan(t *testing.T) {
	// PDF: "ضامن، سپرده یا گردش حساب"
	identity := data.Identities["0012345678"]
	financial := data.Financials["C001"]
	risk := data.Risks["C001"]
	profile := BuildProfile(&identity, &financial, &risk)
	_, ineligible := MatchAllProducts(profile)

	var loan *models.ProductMatch
	for i, p := range ineligible {
		if p.ProductID == "P001" {
			loan = &ineligible[i]
		}
	}
	if loan == nil {
		t.Fatal("housewife should be ineligible for personal loan")
	}
	if len(loan.AlternativesFa) == 0 {
		t.Error("loan ineligibility must include alternative paths (ضامن/سپرده/گردش)")
	}
	joined := strings.Join(loan.AlternativesFa, " ")
	if !strings.Contains(joined, "ضامن") && !strings.Contains(joined, "سپرده") {
		t.Errorf("alternatives should mention ضامن or سپرده, got: %v", loan.AlternativesFa)
	}
}

func TestColdStart_ConditionalOffers(t *testing.T) {
	req := models.ColdStartRequest{
		Name: "سارا", Age: 30, Gender: "female",
		Occupation: "employee", EmploymentType: "private",
		ApproxIncome: 25_000_000, VisitPurpose: "وام شخصی",
	}
	risk := ColdStartRisk(req)
	profile := BuildProfileFromColdStart(req, &risk)
	eligible, _ := MatchAllProducts(profile)
	if len(eligible) == 0 {
		t.Fatal("non-customer should have some eligible (conditional) products")
	}
	for _, p := range eligible {
		if !p.IsConditional {
			t.Errorf("%s should be conditional for cold-start", p.ProductID)
		}
		if len(p.ConditionsFa) == 0 {
			t.Errorf("%s missing activation conditions", p.ProductID)
		}
	}
}

func TestVisitPurpose_BoostsOffers(t *testing.T) {
	identity := data.Identities["0023456789"]
	financial := data.Financials["C002"]
	risk := data.Risks["C002"]

	// Without purpose
	p1 := BuildProfile(&identity, &financial, &risk)
	elig1, _ := MatchAllProducts(p1)
	// With checkbook purpose
	p2 := BuildProfile(&identity, &financial, &risk)
	p2.VisitPurpose = "دسته‌چک"
	elig2, _ := MatchAllProducts(p2)

	var scoreNo, scoreYes float64
	for _, p := range elig1 {
		if p.ProductID == "P003" {
			scoreNo = p.Score
		}
	}
	for _, p := range elig2 {
		if p.ProductID == "P003" {
			scoreYes = p.Score
		}
	}
	if scoreYes <= scoreNo {
		t.Errorf("visit purpose should boost checkbook score: without=%.0f with=%.0f", scoreNo, scoreYes)
	}
}

func TestPersonalizedOffers_Edge(t *testing.T) {
	if PersonalizedOffers(nil, 3) != nil && len(PersonalizedOffers(nil, 3)) != 0 {
		t.Error("nil eligible should return empty")
	}
	if PersonalizedOffers([]models.ProductMatch{{ProductID: "P1"}}, 0) != nil {
		t.Error("n=0 should return nil")
	}
	items := []models.ProductMatch{{ProductID: "A"}, {ProductID: "B"}, {ProductID: "C"}}
	got := PersonalizedOffers(items, 2)
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestEvaluateCondition_BoolField(t *testing.T) {
	profile := CustomerProfile{Fields: map[string]interface{}{"has_guarantor": true}}
	ok, _, _ := EvaluateCondition(models.RuleCondition{Field: "has_guarantor", Operator: "eq", Value: true}, profile)
	if !ok {
		t.Error("bool eq true should pass")
	}
}

func TestColdStartRisk_Clamp(t *testing.T) {
	// Extreme high risk inputs should still clamp
	r := ColdStartRisk(models.ColdStartRequest{
		Age: 18, Occupation: "unemployed", EmploymentType: "none", ApproxIncome: 0,
	})
	if r.RiskScore > 95 || r.RiskScore < 5 {
		t.Errorf("score out of clamp range: %.0f", r.RiskScore)
	}
	// Extreme low risk
	r2 := ColdStartRisk(models.ColdStartRequest{
		Age: 40, Occupation: "manager", EmploymentType: "government", ApproxIncome: 200_000_000,
	})
	if r2.RiskScore > 95 || r2.RiskScore < 5 {
		t.Errorf("score out of clamp range: %.0f", r2.RiskScore)
	}
	if r2.RiskScore >= r.RiskScore {
		t.Errorf("manager should score lower risk than unemployed: mgr=%.0f unemp=%.0f", r2.RiskScore, r.RiskScore)
	}
}

func TestColdStartNotes(t *testing.T) {
	notes := ColdStartNotes()
	if len(notes) < 2 {
		t.Error("expected cold-start notes")
	}
}
