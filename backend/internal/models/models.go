// Package models — مدل‌های داده و قرارداد JSON API
//
// تمام فیلدهای *_fa خروجی فارسی برای کارمند شعبه هستند.
// فیلدهای کلیدی تطبیق (Match):
//
//	ProductMatch.IsConditional / ConditionsFa  → افر مشروط غیرمشتری
//	ProductMatch.AdviceFa / AlternativesFa     → اقدامات و مسیر جایگزین
//	ProductMatch.ObligationsFa / CreditLimitFa → تعهدات و سقف اعتبار
//	MatchResponse.NotesFa / UpstreamErrors     → یادداشت سیستمی و خطای بالادستی
//	MatchResponse.DefaultWarning               → پیامد عدم پرداخت
package models

// Customer identity profile from identity service
type IdentityProfile struct {
	CustomerID      string `json:"customer_id"`
	NationalID      string `json:"national_id"`
	Name            string `json:"name"`
	Age             int    `json:"age"`
	Gender          string `json:"gender"`            // "male" / "female"
	Occupation      string `json:"occupation"`        // "employee", "self_employed", "housewife", "retired", "unemployed", "manager", "student"
	EmploymentType  string `json:"employment_type"`   // "government", "private", "freelance", "none"
	CustomerType    string `json:"customer_type"`     // "real" / "legal"
	AccountOpenDate string `json:"account_open_date"` // ISO date
	IsExisting      bool   `json:"is_existing"`
}

// Financial profile from financial service
type FinancialProfile struct {
	CustomerID         string  `json:"customer_id"`
	MonthlyIncome      float64 `json:"monthly_income"`       // تومان
	AccountTurnover3M  float64 `json:"account_turnover_3m"`  // گردش ۳ ماه اخیر
	AccountTurnover12M float64 `json:"account_turnover_12m"` // گردش ۱۲ ماه اخیر
	TotalDeposits      float64 `json:"total_deposits"`       // مجموع سپرده‌ها
	ActiveLoans        int     `json:"active_loans"`         // تعداد وام‌های فعال
	TotalLoanAmount    float64 `json:"total_loan_amount"`    // مجموع مبالغ وام
	InstallmentDefault int     `json:"installment_default"`  // تعداد اقساط معوق
	SpendingPattern    string  `json:"spending_pattern"`     // "conservative", "moderate", "aggressive"
	PaymentHistory     string  `json:"payment_history"`      // "excellent", "good", "fair", "poor"
	HasGuarantor       bool    `json:"has_guarantor"`
}

// RBCI risk assessment
type RiskAssessment struct {
	CustomerID  string  `json:"customer_id"`
	RiskLevel   string  `json:"risk_level"` // "low", "medium", "high"
	RiskScore   float64 `json:"risk_score"` // 0-100
	Reason      string  `json:"reason"`
	IsColdStart bool    `json:"is_cold_start"` // ارزیابی بر مبنای اطلاعات خوداظهاری
}

// Cold-start risk request for non-customers
type ColdStartRequest struct {
	Name           string  `json:"name"`
	Age            int     `json:"age"`
	Gender         string  `json:"gender"`
	Occupation     string  `json:"occupation"`
	EmploymentType string  `json:"employment_type"`
	ApproxIncome   float64 `json:"approx_income"`
	VisitPurpose   string  `json:"visit_purpose"`
}

// Banking product
type Product struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	NameFa        string `json:"name_fa"`
	Category      string `json:"category"` // "loan", "checkbook", "credit_card", "deposit", "service"
	Description   string `json:"description"`
	DescriptionFa string `json:"description_fa"`
}

// Circular rule — a single eligibility condition for a product
type CircularRule struct {
	ID            string          `json:"id"`
	CircularRef   string          `json:"circular_ref"` // شماره بخشنامه
	CircularRefFa string          `json:"circular_ref_fa"`
	ProductID     string          `json:"product_id"`
	Conditions    []RuleCondition `json:"conditions"`
	Description   string          `json:"description"`
	DescriptionFa string          `json:"description_fa"`
}

// Single condition within a rule
type RuleCondition struct {
	Field    string      `json:"field"`    // "age", "gender", "occupation", "monthly_income", "risk_level", etc.
	Operator string      `json:"operator"` // "eq", "neq", "gt", "gte", "lt", "lte", "in", "not_in"
	Value    interface{} `json:"value"`
}

// Match result for a single product
type ProductMatch struct {
	ProductID      string    `json:"product_id"`
	ProductName    string    `json:"product_name"`
	ProductNameFa  string    `json:"product_name_fa"`
	Eligible       bool      `json:"eligible"`
	IsConditional  bool      `json:"is_conditional,omitempty"` // افر مشروط (غیرمشتری)
	ConditionsFa   []string  `json:"conditions_fa,omitempty"`  // شروط فعال‌سازی افر
	Reasons        []string  `json:"reasons"`
	ReasonsFa      []string  `json:"reasons_fa"`
	Gaps           []GapItem `json:"gaps,omitempty"`
	AdviceFa       []string  `json:"advice_fa,omitempty"`       // اقدامات عملی برای مجاز شدن
	AlternativesFa []string  `json:"alternatives_fa,omitempty"` // مسیرهای جایگزین (ضامن، سپرده، ...)
	ObligationsFa  []string  `json:"obligations_fa,omitempty"`  // تعهدات و الزامات پس از دریافت
	CreditLimitFa  string    `json:"credit_limit_fa,omitempty"` // سقف اعتبار
	CircularRefs   []string  `json:"circular_refs"`
	Score          float64   `json:"score"`
}

// Gap analysis item — what needs to change for eligibility
type GapItem struct {
	Field         string `json:"field"`
	CurrentVal    string `json:"current_value"`
	RequiredVal   string `json:"required_value"`
	Description   string `json:"description"`
	DescriptionFa string `json:"description_fa"`
	AdviceFa      string `json:"advice_fa,omitempty"` // اقدام عملی برای رفع این شکاف
}

// Full match response
type MatchResponse struct {
	CustomerID         string          `json:"customer_id"`
	NationalID         string          `json:"national_id"`
	CustomerName       string          `json:"customer_name"`
	IsExisting         bool            `json:"is_existing"`
	IsColdStart        bool            `json:"is_cold_start"`
	RiskLevel          string          `json:"risk_level"`
	RiskScore          float64         `json:"risk_score"`
	RiskReason         string          `json:"risk_reason,omitempty"`
	VisitPurpose       string          `json:"visit_purpose,omitempty"`
	EligibleProducts   []ProductMatch  `json:"eligible_products"`
	IneligibleProducts []ProductMatch  `json:"ineligible_products"`
	PersonalizedOffers []ProductMatch  `json:"personalized_offers"`
	NotesFa            []string        `json:"notes_fa,omitempty"` // یادداشت‌های سیستمی (افر مشروط و ...)
	DefaultWarning     *DefaultWarning `json:"default_warning,omitempty"`
	UpstreamErrors     []string        `json:"upstream_errors,omitempty"` // خطاهای سامانه‌های بالادستی
}

// CustomerRecord is the local RBCI endpoint payload.
// It stores identity, financial behaviour, and RBCI risk data in PostgreSQL.
type CustomerRecord struct {
	Identity  IdentityProfile  `json:"identity"`
	Financial FinancialProfile `json:"financial"`
	Risk      RiskAssessment   `json:"risk"`
}

// Warning about payment default consequences
type DefaultWarning struct {
	CurrentRiskLevel   string   `json:"current_risk_level"`
	PotentialRiskLevel string   `json:"potential_risk_level"`
	Consequences       []string `json:"consequences"`
	ConsequencesFa     []string `json:"consequences_fa"`
}
