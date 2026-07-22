// Package data — داده‌های اولیه endpoint محلی RBCI و قواعد بانکی
//
// شامل:
//
//	Identities  — ۵ مشتری seed (کد ملی ۱۰ رقمی)
//	Financials  — پروفایل مالی seed
//	Risks       — ارزیابی RBCI seed
//	Products    — ۱۰ محصول P001..P010
//	Circulars   — قوانین اهلیت استخراج‌شده از بخشنامه‌ها
//
// سناریوهای PDF:
//
//	C001/0012345678 خانه‌دار ۴۰ ساله (درآمد پایین، گردش کم)
//	C002/0023456789 مدیر پردرآمد (واجد دسته‌چک و وام)
//	C003/0034567890 کارمند با اقساط معوق
package data

import "github.com/banking-assistant/backend/internal/models"

// Seed identity profiles
var Identities = map[string]models.IdentityProfile{
	"0012345678": {
		CustomerID: "C001", NationalID: "0012345678",
		Name: "فاطمه احمدی", Age: 40, Gender: "female",
		Occupation: "housewife", EmploymentType: "none",
		CustomerType: "real", AccountOpenDate: "1400/06/15", IsExisting: true,
	},
	"0023456789": {
		CustomerID: "C002", NationalID: "0023456789",
		Name: "علی رضایی", Age: 45, Gender: "male",
		Occupation: "manager", EmploymentType: "private",
		CustomerType: "real", AccountOpenDate: "1395/01/10", IsExisting: true,
	},
	"0034567890": {
		CustomerID: "C003", NationalID: "0034567890",
		Name: "محمد حسینی", Age: 32, Gender: "male",
		Occupation: "employee", EmploymentType: "government",
		CustomerType: "real", AccountOpenDate: "1401/03/20", IsExisting: true,
	},
	"0045678901": {
		CustomerID: "C004", NationalID: "0045678901",
		Name: "زهرا کریمی", Age: 28, Gender: "female",
		Occupation: "employee", EmploymentType: "private",
		CustomerType: "real", AccountOpenDate: "1402/07/01", IsExisting: true,
	},
	"0056789012": {
		CustomerID: "C005", NationalID: "0056789012",
		Name: "رضا محمدی", Age: 55, Gender: "male",
		Occupation: "retired", EmploymentType: "none",
		CustomerType: "real", AccountOpenDate: "1385/11/05", IsExisting: true,
	},
}

// Seed financial profiles
var Financials = map[string]models.FinancialProfile{
	"C001": {
		CustomerID: "C001", MonthlyIncome: 8_000_000,
		AccountTurnover3M: 30_000_000, AccountTurnover12M: 100_000_000,
		TotalDeposits: 50_000_000, ActiveLoans: 0, TotalLoanAmount: 0,
		InstallmentDefault: 0, SpendingPattern: "conservative",
		PaymentHistory: "good", HasGuarantor: false,
	},
	"C002": {
		CustomerID: "C002", MonthlyIncome: 120_000_000,
		AccountTurnover3M: 500_000_000, AccountTurnover12M: 2_000_000_000,
		TotalDeposits: 800_000_000, ActiveLoans: 1, TotalLoanAmount: 300_000_000,
		InstallmentDefault: 0, SpendingPattern: "moderate",
		PaymentHistory: "excellent", HasGuarantor: true,
	},
	"C003": {
		CustomerID: "C003", MonthlyIncome: 35_000_000,
		AccountTurnover3M: 120_000_000, AccountTurnover12M: 500_000_000,
		TotalDeposits: 150_000_000, ActiveLoans: 1, TotalLoanAmount: 100_000_000,
		InstallmentDefault: 2, SpendingPattern: "moderate",
		PaymentHistory: "fair", HasGuarantor: false,
	},
	"C004": {
		CustomerID: "C004", MonthlyIncome: 25_000_000,
		AccountTurnover3M: 80_000_000, AccountTurnover12M: 350_000_000,
		TotalDeposits: 60_000_000, ActiveLoans: 0, TotalLoanAmount: 0,
		InstallmentDefault: 0, SpendingPattern: "conservative",
		PaymentHistory: "good", HasGuarantor: false,
	},
	"C005": {
		CustomerID: "C005", MonthlyIncome: 15_000_000,
		AccountTurnover3M: 50_000_000, AccountTurnover12M: 200_000_000,
		TotalDeposits: 300_000_000, ActiveLoans: 0, TotalLoanAmount: 0,
		InstallmentDefault: 0, SpendingPattern: "conservative",
		PaymentHistory: "excellent", HasGuarantor: false,
	},
}

// Seed risk assessments
var Risks = map[string]models.RiskAssessment{
	"C001": {CustomerID: "C001", RiskLevel: "medium", RiskScore: 55, Reason: "درآمد ثابت اندک، بدون سابقه وام"},
	"C002": {CustomerID: "C002", RiskLevel: "low", RiskScore: 15, Reason: "درآمد بالا، سابقه پرداخت عالی، گردش حساب بالا"},
	"C003": {CustomerID: "C003", RiskLevel: "medium", RiskScore: 50, Reason: "اقساط معوق، سابقه پرداخت متوسط"},
	"C004": {CustomerID: "C004", RiskLevel: "low", RiskScore: 25, Reason: "بدون وام فعال، سابقه پرداخت خوب"},
	"C005": {CustomerID: "C005", RiskLevel: "low", RiskScore: 10, Reason: "سابقه طولانی، بدون بدهی، سپرده بالا"},
}

// Banking products
var Products = []models.Product{
	{ID: "P001", Name: "Personal Loan", NameFa: "وام شخصی", Category: "loan",
		Description: "Personal loan up to 500M Toman", DescriptionFa: "وام شخصی تا سقف ۵۰۰ میلیون تومان"},
	{ID: "P002", Name: "Housing Loan", NameFa: "وام مسکن", Category: "loan",
		Description: "Housing loan up to 2B Toman", DescriptionFa: "وام مسکن تا سقف ۲ میلیارد تومان"},
	{ID: "P003", Name: "Checkbook", NameFa: "دسته‌چک", Category: "checkbook",
		Description: "Standard checkbook issuance", DescriptionFa: "صدور دسته‌چک عادی"},
	{ID: "P004", Name: "Credit Card", NameFa: "کارت اعتباری", Category: "credit_card",
		Description: "Credit card with revolving limit", DescriptionFa: "کارت اعتباری با اعتبار گردشی"},
	{ID: "P005", Name: "Short-term Deposit", NameFa: "سپرده کوتاه‌مدت", Category: "deposit",
		Description: "3-month short-term deposit", DescriptionFa: "سپرده کوتاه‌مدت ۳ ماهه"},
	{ID: "P006", Name: "Long-term Deposit", NameFa: "سپرده بلندمدت", Category: "deposit",
		Description: "1-year long-term deposit", DescriptionFa: "سپرده بلندمدت یک‌ساله"},
	{ID: "P007", Name: "Business Loan", NameFa: "وام کسب‌وکار", Category: "loan",
		Description: "Business loan for self-employed", DescriptionFa: "وام کسب‌وکار برای صاحبان مشاغل"},
	{ID: "P008", Name: "Online Banking", NameFa: "بانکداری اینترنتی", Category: "service",
		Description: "Internet banking activation", DescriptionFa: "فعال‌سازی بانکداری اینترنتی"},
	{ID: "P009", Name: "SMS Banking", NameFa: "پیامک بانکی", Category: "service",
		Description: "SMS notification service", DescriptionFa: "سرویس اطلاع‌رسانی پیامکی"},
	{ID: "P010", Name: "Marriage Loan", NameFa: "وام ازدواج", Category: "loan",
		Description: "Marriage loan for eligible couples", DescriptionFa: "وام ازدواج برای زوجین واجد شرایط"},
}

// Circular rules — eligibility conditions per product
var Circulars = []models.CircularRule{
	// === وام شخصی P001 ===
	{
		ID: "R001", CircularRef: "BN-1404/123", CircularRefFa: "بخشنامه شماره ۱۴۰۴/۱۲۳",
		ProductID: "P001",
		Conditions: []models.RuleCondition{
			{Field: "age", Operator: "gte", Value: float64(18)},
			{Field: "age", Operator: "lte", Value: float64(65)},
			{Field: "risk_level", Operator: "in", Value: []interface{}{"low", "medium"}},
			{Field: "monthly_income", Operator: "gte", Value: float64(10_000_000)},
			{Field: "installment_default", Operator: "lte", Value: float64(1)},
		},
		Description:   "Personal loan eligibility: age 18-65, low/medium risk, income >= 10M, max 1 default",
		DescriptionFa: "شرایط اهلیت وام شخصی: سن ۱۸ تا ۶۵ سال، ریسک کم یا متوسط، درآمد حداقل ۱۰ میلیون، حداکثر ۱ قسط معوق",
	},
	// === وام مسکن P002 ===
	{
		ID: "R002", CircularRef: "BN-1404/124", CircularRefFa: "بخشنامه شماره ۱۴۰۴/۱۲۴",
		ProductID: "P002",
		Conditions: []models.RuleCondition{
			{Field: "age", Operator: "gte", Value: float64(21)},
			{Field: "age", Operator: "lte", Value: float64(60)},
			{Field: "risk_level", Operator: "eq", Value: "low"},
			{Field: "monthly_income", Operator: "gte", Value: float64(30_000_000)},
			{Field: "account_turnover_12m", Operator: "gte", Value: float64(500_000_000)},
			{Field: "installment_default", Operator: "eq", Value: float64(0)},
			{Field: "payment_history", Operator: "in", Value: []interface{}{"excellent", "good"}},
		},
		Description:   "Housing loan: age 21-60, low risk only, income >= 30M, 12M turnover >= 500M, no defaults, good+ payment history",
		DescriptionFa: "شرایط اهلیت وام مسکن: سن ۲۱ تا ۶۰، فقط ریسک کم، درآمد حداقل ۳۰ میلیون، گردش ۱۲ ماه حداقل ۵۰۰ میلیون، بدون قسط معوق، سابقه پرداخت خوب یا عالی",
	},
	// === دسته‌چک P003 ===
	{
		ID: "R003", CircularRef: "BN-1404/125", CircularRefFa: "بخشنامه شماره ۱۴۰۴/۱۲۵",
		ProductID: "P003",
		Conditions: []models.RuleCondition{
			{Field: "age", Operator: "gte", Value: float64(18)},
			{Field: "risk_level", Operator: "in", Value: []interface{}{"low", "medium"}},
			{Field: "account_turnover_3m", Operator: "gte", Value: float64(100_000_000)},
			{Field: "installment_default", Operator: "eq", Value: float64(0)},
			{Field: "occupation", Operator: "not_in", Value: []interface{}{"unemployed", "student"}},
		},
		Description:   "Checkbook: age 18+, low/medium risk, 3M turnover >= 100M, no defaults, not unemployed/student",
		DescriptionFa: "شرایط اهلیت دسته‌چک: سن ۱۸+، ریسک کم یا متوسط، گردش ۳ ماه حداقل ۱۰۰ میلیون، بدون قسط معوق، شاغل",
	},
	// === کارت اعتباری P004 ===
	{
		ID: "R004", CircularRef: "BN-1404/126", CircularRefFa: "بخشنامه شماره ۱۴۰۴/۱۲۶",
		ProductID: "P004",
		Conditions: []models.RuleCondition{
			{Field: "age", Operator: "gte", Value: float64(18)},
			{Field: "age", Operator: "lte", Value: float64(70)},
			{Field: "risk_level", Operator: "in", Value: []interface{}{"low", "medium"}},
			{Field: "monthly_income", Operator: "gte", Value: float64(15_000_000)},
			{Field: "installment_default", Operator: "lte", Value: float64(2)},
		},
		Description:   "Credit card: age 18-70, low/medium risk, income >= 15M, max 2 defaults",
		DescriptionFa: "شرایط اهلیت کارت اعتباری: سن ۱۸ تا ۷۰، ریسک کم یا متوسط، درآمد حداقل ۱۵ میلیون، حداکثر ۲ قسط معوق",
	},
	// === سپرده کوتاه‌مدت P005 ===
	{
		ID: "R005", CircularRef: "BN-1404/127", CircularRefFa: "بخشنامه شماره ۱۴۰۴/۱۲۷",
		ProductID: "P005",
		Conditions: []models.RuleCondition{
			{Field: "age", Operator: "gte", Value: float64(18)},
		},
		Description:   "Short-term deposit: age 18+ only",
		DescriptionFa: "شرایط سپرده کوتاه‌مدت: فقط سن ۱۸ سال به بالا",
	},
	// === سپرده بلندمدت P006 ===
	{
		ID: "R006", CircularRef: "BN-1404/128", CircularRefFa: "بخشنامه شماره ۱۴۰۴/۱۲۸",
		ProductID: "P006",
		Conditions: []models.RuleCondition{
			{Field: "age", Operator: "gte", Value: float64(18)},
			{Field: "monthly_income", Operator: "gte", Value: float64(5_000_000)},
		},
		Description:   "Long-term deposit: age 18+, income >= 5M",
		DescriptionFa: "شرایط سپرده بلندمدت: سن ۱۸+، درآمد حداقل ۵ میلیون",
	},
	// === وام کسب‌وکار P007 ===
	{
		ID: "R007", CircularRef: "BN-1404/129", CircularRefFa: "بخشنامه شماره ۱۴۰۴/۱۲۹",
		ProductID: "P007",
		Conditions: []models.RuleCondition{
			{Field: "age", Operator: "gte", Value: float64(21)},
			{Field: "age", Operator: "lte", Value: float64(60)},
			{Field: "occupation", Operator: "in", Value: []interface{}{"self_employed", "manager"}},
			{Field: "risk_level", Operator: "eq", Value: "low"},
			{Field: "monthly_income", Operator: "gte", Value: float64(50_000_000)},
			{Field: "account_turnover_3m", Operator: "gte", Value: float64(200_000_000)},
			{Field: "installment_default", Operator: "eq", Value: float64(0)},
		},
		Description:   "Business loan: age 21-60, self-employed/manager, low risk, income >= 50M, 3M turnover >= 200M, no defaults",
		DescriptionFa: "شرایط وام کسب‌وکار: سن ۲۱-۶۰، شغل آزاد یا مدیر، ریسک کم، درآمد ۵۰+ میلیون، گردش ۳ ماهه ۲۰۰+ میلیون، بدون قسط معوق",
	},
	// === بانکداری اینترنتی P008 ===
	{
		ID: "R008", CircularRef: "BN-1404/130", CircularRefFa: "بخشنامه شماره ۱۴۰۴/۱۳۰",
		ProductID: "P008",
		Conditions: []models.RuleCondition{
			{Field: "age", Operator: "gte", Value: float64(15)},
		},
		Description:   "Online banking: age 15+",
		DescriptionFa: "بانکداری اینترنتی: سن ۱۵ سال به بالا",
	},
	// === پیامک بانکی P009 ===
	{
		ID: "R009", CircularRef: "BN-1404/131", CircularRefFa: "بخشنامه شماره ۱۴۰۴/۱۳۱",
		ProductID: "P009",
		Conditions: []models.RuleCondition{
			{Field: "age", Operator: "gte", Value: float64(15)},
		},
		Description:   "SMS banking: age 15+",
		DescriptionFa: "پیامک بانکی: سن ۱۵ سال به بالا",
	},
	// === وام ازدواج P010 ===
	{
		ID: "R010", CircularRef: "BN-1404/132", CircularRefFa: "بخشنامه شماره ۱۴۰۴/۱۳۲",
		ProductID: "P010",
		Conditions: []models.RuleCondition{
			{Field: "age", Operator: "gte", Value: float64(18)},
			{Field: "age", Operator: "lte", Value: float64(40)},
			{Field: "risk_level", Operator: "in", Value: []interface{}{"low", "medium"}},
			{Field: "installment_default", Operator: "eq", Value: float64(0)},
		},
		Description:   "Marriage loan: age 18-40, low/medium risk, no defaults",
		DescriptionFa: "شرایط وام ازدواج: سن ۱۸ تا ۴۰، ریسک کم یا متوسط، بدون قسط معوق",
	},
}

// ProductMap for quick lookup by ID
func ProductMap() map[string]models.Product {
	m := make(map[string]models.Product, len(Products))
	for _, p := range Products {
		m[p.ID] = p
	}
	return m
}

// CircularsByProduct groups rules by product ID
func CircularsByProduct() map[string][]models.CircularRule {
	m := make(map[string][]models.CircularRule)
	for _, c := range Circulars {
		m[c.ProductID] = append(m[c.ProductID], c)
	}
	return m
}
