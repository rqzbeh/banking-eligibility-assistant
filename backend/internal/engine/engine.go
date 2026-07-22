// Package engine — موتور قطعی (deterministic) تعیین اهلیت و پیشنهاد محصول
//
// این بسته هسته منطق کسب‌وکار چالش است:
//   - BuildProfile / BuildProfileFromColdStart: ساخت پروفایل تخت از منابع داده
//   - EvaluateCondition: ارزیابی یک شرط بخشنامه (eq/neq/gt/gte/lt/lte/in/not_in)
//   - MatchAllProducts: تطبیق همه محصولات → eligible / ineligible + score
//   - Gap Analysis + AdviceFa + AlternativesFa (ضامن/سپرده/گردش — سناریوی PDF)
//   - ObligationsFa + CreditLimitFa برای محصولات مجاز
//   - افر مشروط غیرمشتری (IsConditional / ConditionsFa)
//   - visitPurposeBoost: رتبه‌بندی بر اساس هدف مراجعه
//   - ColdStartRisk / GenerateDefaultWarning / ValidateNationalID
//
// هیچ فراخوانی LLM اینجا نیست؛ خروجی همیشه قابل بازتولید است.
package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/banking-assistant/backend/internal/data"
	"github.com/banking-assistant/backend/internal/models"
)

// CustomerProfile flattens identity + financial + risk into a single map for rule evaluation
type CustomerProfile struct {
	Fields       map[string]interface{}
	Identity     *models.IdentityProfile
	Financial    *models.FinancialProfile
	Risk         *models.RiskAssessment
	IsColdStart  bool
	VisitPurpose string
}

// MatchOptions controls matching behaviour
type MatchOptions struct {
	IsColdStart  bool
	VisitPurpose string
}

// BuildProfile constructs a flat profile from the three data sources
func BuildProfile(id *models.IdentityProfile, fin *models.FinancialProfile, risk *models.RiskAssessment) CustomerProfile {
	p := CustomerProfile{
		Fields:   make(map[string]interface{}),
		Identity: id, Financial: fin, Risk: risk,
	}
	if id != nil {
		p.Fields["age"] = float64(id.Age)
		p.Fields["gender"] = id.Gender
		p.Fields["occupation"] = id.Occupation
		p.Fields["employment_type"] = id.EmploymentType
		p.Fields["customer_type"] = id.CustomerType
	}
	if fin != nil {
		p.Fields["monthly_income"] = fin.MonthlyIncome
		p.Fields["account_turnover_3m"] = fin.AccountTurnover3M
		p.Fields["account_turnover_12m"] = fin.AccountTurnover12M
		p.Fields["total_deposits"] = fin.TotalDeposits
		p.Fields["active_loans"] = float64(fin.ActiveLoans)
		p.Fields["total_loan_amount"] = fin.TotalLoanAmount
		p.Fields["installment_default"] = float64(fin.InstallmentDefault)
		p.Fields["spending_pattern"] = fin.SpendingPattern
		p.Fields["payment_history"] = fin.PaymentHistory
		p.Fields["has_guarantor"] = fin.HasGuarantor
	}
	if risk != nil {
		p.Fields["risk_level"] = risk.RiskLevel
		p.Fields["risk_score"] = risk.RiskScore
	}
	return p
}

// BuildProfileFromColdStart creates a profile from self-declared data
func BuildProfileFromColdStart(req models.ColdStartRequest, risk *models.RiskAssessment) CustomerProfile {
	p := CustomerProfile{
		Fields:       make(map[string]interface{}),
		Risk:         risk,
		IsColdStart:  true,
		VisitPurpose: req.VisitPurpose,
	}
	p.Fields["age"] = float64(req.Age)
	p.Fields["gender"] = req.Gender
	p.Fields["occupation"] = req.Occupation
	p.Fields["employment_type"] = req.EmploymentType
	p.Fields["monthly_income"] = req.ApproxIncome
	// Non-customers have no financial history
	p.Fields["account_turnover_3m"] = float64(0)
	p.Fields["account_turnover_12m"] = float64(0)
	p.Fields["total_deposits"] = float64(0)
	p.Fields["active_loans"] = float64(0)
	p.Fields["total_loan_amount"] = float64(0)
	p.Fields["installment_default"] = float64(0)
	p.Fields["spending_pattern"] = "unknown"
	p.Fields["payment_history"] = "unknown"
	p.Fields["has_guarantor"] = false
	if risk != nil {
		p.Fields["risk_level"] = risk.RiskLevel
		p.Fields["risk_score"] = risk.RiskScore
	}
	return p
}

// ValidateNationalID checks Iranian national ID format (10 digits, not all zeros)
func ValidateNationalID(nid string) (bool, string) {
	nid = strings.TrimSpace(nid)
	if nid == "" {
		return false, "کد ملی الزامی است"
	}
	if len(nid) != 10 {
		return false, "کد ملی باید ۱۰ رقم باشد"
	}
	allZero := true
	for _, c := range nid {
		if c < '0' || c > '9' {
			return false, "کد ملی باید فقط شامل ارقام باشد"
		}
		if c != '0' {
			allZero = false
		}
	}
	if allZero {
		return false, "کد ملی نامعتبر است"
	}
	return true, ""
}

// NormalizeOccupation maps common Persian/English occupation aliases
func NormalizeOccupation(occ string) string {
	occ = strings.TrimSpace(strings.ToLower(occ))
	aliases := map[string]string{
		"خانه دار": "housewife", "خانه‌دار": "housewife", "house wife": "housewife",
		"کارمند": "employee", "کارگر": "employee",
		"مدیر": "manager", "مدیربانک": "manager",
		"بازنشسته": "retired", "بازنشسته/مستمری‌بگیر": "retired",
		"بیکار": "unemployed", "جویای کار": "unemployed",
		"دانشجو": "student", "محصل": "student",
		"آزاد": "self_employed", "شغل آزاد": "self_employed", "کاسب": "self_employed",
		"فریلنسر": "self_employed", "freelance": "self_employed",
	}
	if v, ok := aliases[occ]; ok {
		return v
	}
	return occ
}

// EvaluateCondition checks a single condition against the profile
func EvaluateCondition(cond models.RuleCondition, profile CustomerProfile) (bool, string, string) {
	val, exists := profile.Fields[cond.Field]
	if !exists {
		reason := fmt.Sprintf("فیلد %s در پروفایل مشتری موجود نیست", cond.Field)
		return false, fmt.Sprintf("Field '%s' not found in profile", cond.Field), reason
	}

	switch cond.Operator {
	case "eq":
		ok := fmt.Sprintf("%v", val) == fmt.Sprintf("%v", cond.Value)
		if !ok {
			return false,
				fmt.Sprintf("%s must be %v (current: %v)", cond.Field, cond.Value, val),
				fmt.Sprintf("%s باید %v باشد (فعلی: %v)", fieldNameFa(cond.Field), cond.Value, val)
		}
		return true, "", ""

	case "neq":
		ok := fmt.Sprintf("%v", val) != fmt.Sprintf("%v", cond.Value)
		if !ok {
			return false,
				fmt.Sprintf("%s must not be %v", cond.Field, cond.Value),
				fmt.Sprintf("%s نباید %v باشد", fieldNameFa(cond.Field), cond.Value)
		}
		return true, "", ""

	case "gt", "gte", "lt", "lte":
		numVal := toFloat(val)
		numCond := toFloat(cond.Value)
		var ok bool
		switch cond.Operator {
		case "gt":
			ok = numVal > numCond
		case "gte":
			ok = numVal >= numCond
		case "lt":
			ok = numVal < numCond
		case "lte":
			ok = numVal <= numCond
		}
		if !ok {
			opFa := operatorFa(cond.Operator)
			return false,
				fmt.Sprintf("%s must be %s %v (current: %v)", cond.Field, cond.Operator, numCond, numVal),
				fmt.Sprintf("%s باید %s %s باشد (فعلی: %s)", fieldNameFa(cond.Field), opFa, formatNumber(numCond), formatNumber(numVal))
		}
		return true, "", ""

	case "in":
		list, ok := cond.Value.([]interface{})
		if !ok {
			return false, "Invalid 'in' value", "مقدار نامعتبر"
		}
		strVal := fmt.Sprintf("%v", val)
		for _, item := range list {
			if fmt.Sprintf("%v", item) == strVal {
				return true, "", ""
			}
		}
		return false,
			fmt.Sprintf("%s must be one of %v (current: %v)", cond.Field, list, val),
			fmt.Sprintf("%s باید یکی از %v باشد (فعلی: %v)", fieldNameFa(cond.Field), list, val)

	case "not_in":
		list, ok := cond.Value.([]interface{})
		if !ok {
			return false, "Invalid 'not_in' value", "مقدار نامعتبر"
		}
		strVal := fmt.Sprintf("%v", val)
		for _, item := range list {
			if fmt.Sprintf("%v", item) == strVal {
				return false,
					fmt.Sprintf("%s must not be one of %v (current: %v)", cond.Field, list, val),
					fmt.Sprintf("%s نباید یکی از %v باشد (فعلی: %v)", fieldNameFa(cond.Field), list, val)
			}
		}
		return true, "", ""
	}

	return false, "Unknown operator", "عملگر ناشناخته"
}

// MatchAllProducts evaluates all products against the customer profile
func MatchAllProducts(profile CustomerProfile) ([]models.ProductMatch, []models.ProductMatch) {
	productMap := data.ProductMap()
	circularsByProduct := data.CircularsByProduct()

	var eligible, ineligible []models.ProductMatch

	for _, product := range data.Products {
		rules := circularsByProduct[product.ID]
		if len(rules) == 0 {
			// No rules = eligible by default (basic services)
			m := models.ProductMatch{
				ProductID: product.ID, ProductName: product.Name, ProductNameFa: product.NameFa,
				Eligible: true, Reasons: []string{"No restrictions"}, ReasonsFa: []string{"بدون محدودیت"},
				Score: calculateScore(product, profile),
			}
			if profile.IsColdStart {
				m.IsConditional = true
				m.ConditionsFa = coldStartConditions(product.ID)
				m.ReasonsFa = []string{"مجاز مشروط برای مشتری جدید (بدون سابقه بانکی)"}
			}
			eligible = append(eligible, m)
			continue
		}

		allPassed := true
		var failReasons, failReasonsFa []string
		var gaps []models.GapItem
		var refs []string

		for _, rule := range rules {
			refs = append(refs, rule.CircularRef)
			for _, cond := range rule.Conditions {
				passed, reason, reasonFa := EvaluateCondition(cond, profile)
				if !passed {
					allPassed = false
					failReasons = append(failReasons, reason)
					failReasonsFa = append(failReasonsFa, reasonFa)

					currentVal := profile.Fields[cond.Field]
					gaps = append(gaps, models.GapItem{
						Field:         cond.Field,
						CurrentVal:    fmt.Sprintf("%v", currentVal),
						RequiredVal:   fmt.Sprintf("%s %v", cond.Operator, cond.Value),
						Description:   reason,
						DescriptionFa: reasonFa,
						AdviceFa:      adviceForGap(cond, currentVal),
					})
				}
			}
		}

		_ = productMap
		match := models.ProductMatch{
			ProductID: product.ID, ProductName: product.Name, ProductNameFa: product.NameFa,
			Eligible: allPassed, CircularRefs: refs,
		}

		if allPassed {
			match.Reasons = []string{"All conditions met"}
			match.ReasonsFa = []string{"تمام شرایط رعایت شده است"}
			match.Score = calculateScore(product, profile)
			match.ObligationsFa = productObligations(product.ID, profile)
			match.CreditLimitFa = productCreditLimit(product.ID, profile)
			if profile.IsColdStart {
				// PDF: non-customer offers can be conditional (account opening / docs)
				match.IsConditional = true
				match.ConditionsFa = coldStartConditions(product.ID)
				match.ReasonsFa = []string{"اهلیت اولیه تأیید شد — فعال‌سازی منوط به افتتاح حساب و ارائه مدارک تکمیلی"}
			}
			eligible = append(eligible, match)
		} else {
			match.Reasons = failReasons
			match.ReasonsFa = failReasonsFa
			match.Gaps = gaps
			match.AdviceFa = aggregateAdvice(gaps)
			match.AlternativesFa = alternativePaths(product.ID, gaps, profile)
			match.Score = 0
			ineligible = append(ineligible, match)
		}
	}

	// Sort eligible by score descending
	sort.Slice(eligible, func(i, j int) bool {
		return eligible[i].Score > eligible[j].Score
	})

	return eligible, ineligible
}

// coldStartConditions returns activation conditions for non-customer offers
func coldStartConditions(productID string) []string {
	base := []string{
		"افتتاح حساب نزد بانک",
		"ارائه مدارک هویتی و شغلی معتبر",
		"تأیید نهایی ارزیابی ریسک پس از تشکیل پرونده",
	}
	switch productID {
	case "P001", "P002", "P007", "P010":
		return append(base, "تکمیل پرونده اعتباری و در صورت نیاز معرفی ضامن")
	case "P003":
		return append(base, "ایجاد گردش حساب حداقل ۳ ماهه پس از افتتاح حساب")
	case "P004":
		return append(base, "تعیین سقف اعتبار پس از بررسی درآمد مستند")
	default:
		return base
	}
}

// alternativePaths suggests guarantor/deposit/other routes when ineligible
// PDF housewife scenario: "ضامن، سپرده یا گردش حساب"
func alternativePaths(productID string, gaps []models.GapItem, profile CustomerProfile) []string {
	var alts []string
	fields := map[string]bool{}
	for _, g := range gaps {
		fields[g.Field] = true
	}

	switch productID {
	case "P001", "P002", "P007", "P010": // loans
		if fields["monthly_income"] || fields["risk_level"] || fields["installment_default"] {
			alts = append(alts, "معرفی ضامن واجد شرایط (کارمند رسمی با فیش حقوقی) می‌تواند مسیر تسهیلات را باز کند")
		}
		if fields["monthly_income"] || fields["account_turnover_3m"] || fields["account_turnover_12m"] {
			alts = append(alts, "افتتاح/افزایش سپرده بلندمدت به‌عنوان وثیقه نقدی (حداقل معادل ۲۰٪ مبلغ تسهیلات)")
		}
		if fields["account_turnover_3m"] || fields["account_turnover_12m"] {
			alts = append(alts, "تمرکز واریز حقوق و تراکنش‌ها در همین حساب به‌مدت حداقل ۳ ماه برای ساخت گردش")
		}
		if hasGuarantor, _ := profile.Fields["has_guarantor"].(bool); hasGuarantor {
			alts = append(alts, "ضامن در پروفایل ثبت شده — پرونده را با مدارک ضامن به شعبه ارسال کنید")
		}
	case "P003": // checkbook
		if fields["installment_default"] {
			alts = append(alts, "تسویه کامل اقساط معوق و انتظار ۳ ماه بدون معوقه — پیش‌نیاز صدور دسته‌چک")
		}
		if fields["account_turnover_3m"] {
			alts = append(alts, "افزایش گردش حساب ۳ ماهه از طریق واریز مستمر حقوق/درآمد به همین حساب")
		}
		if fields["occupation"] {
			alts = append(alts, "ارائه مدارک اشتغال معتبر برای به‌روزرسانی وضعیت شغلی")
		}
	case "P004":
		if fields["monthly_income"] {
			alts = append(alts, "ارائه گواهی درآمد تکمیلی یا فیش حقوقی به‌روز")
		}
		if fields["installment_default"] {
			alts = append(alts, "کاهش تعداد اقساط معوق به حداکثر ۲ مورد")
		}
	}

	// Always offer basic services as soft alternative when credit products fail
	if productID == "P001" || productID == "P002" || productID == "P003" || productID == "P007" {
		alts = append(alts, "در حال حاضر می‌توانید از سپرده کوتاه‌مدت، بانکداری اینترنتی و پیامک بانکی استفاده کنید")
	}
	return alts
}

// adviceForGap returns actionable Farsi advice for a failed condition
func adviceForGap(cond models.RuleCondition, currentVal interface{}) string {
	switch cond.Field {
	case "monthly_income":
		needed := toFloat(cond.Value)
		current := toFloat(currentVal)
		diff := needed - current
		if diff > 0 {
			return fmt.Sprintf("درآمد ماهانه را حداقل %s افزایش دهید (از طریق ارتقای شغلی، درآمد جانبی یا ارائه گواهی درآمد بالاتر)", formatNumber(diff))
		}
	case "account_turnover_3m":
		needed := toFloat(cond.Value)
		current := toFloat(currentVal)
		diff := needed - current
		if diff > 0 {
			return fmt.Sprintf("گردش حساب ۳ ماهه را %s افزایش دهید — واریز حقوق/درآمد به همین حساب و تجمیع تراکنش‌ها در ۳ ماه آینده", formatNumber(diff))
		}
		return "تمام درآمد و تراکنش‌های مالی را از همین حساب بانکی انجام دهید تا گردش حساب افزایش یابد"
	case "account_turnover_12m":
		needed := toFloat(cond.Value)
		current := toFloat(currentVal)
		diff := needed - current
		if diff > 0 {
			return fmt.Sprintf("گردش حساب ۱۲ ماهه را %s افزایش دهید — تمرکز تراکنش‌ها در همین حساب طی ماه‌های آینده", formatNumber(diff))
		}
	case "installment_default":
		current := toFloat(currentVal)
		if current > 0 {
			return fmt.Sprintf("ابتدا %s قسط معوق را تسویه کنید، سپس حداقل ۳ ماه سابقه پرداخت منظم ایجاد کنید", formatNumber(current))
		}
	case "risk_level":
		return "با تسویه بدهی‌ها، افزایش گردش حساب و بهبود سابقه پرداخت، سطح ریسک را کاهش دهید"
	case "payment_history":
		return "حداقل ۶ ماه پرداخت به‌موقع اقساط و تعهدات داشته باشید تا سابقه پرداخت بهبود یابد"
	case "occupation":
		return "با ارائه مدارک شغلی معتبر (حکم کارگزینی، پروانه کسب یا گواهی اشتغال) وضعیت شغلی را به‌روزرسانی کنید"
	case "age":
		return "این شرط سنی است و قابل تغییر نیست — محصول جایگزین پیشنهاد شود"
	case "has_guarantor":
		return "یک ضامن واجد شرایط (کارمند رسمی با فیش حقوقی) معرفی کنید"
	case "total_deposits":
		needed := toFloat(cond.Value)
		return fmt.Sprintf("موجودی سپرده را به حداقل %s برسانید", formatNumber(needed))
	}
	return fmt.Sprintf("شرط %s را مطابق بخشنامه برآورده کنید", fieldNameFa(cond.Field))
}

// aggregateAdvice deduplicates gap advice into a product-level list
func aggregateAdvice(gaps []models.GapItem) []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range gaps {
		if g.AdviceFa != "" && !seen[g.AdviceFa] {
			seen[g.AdviceFa] = true
			out = append(out, g.AdviceFa)
		}
	}
	return out
}

// productObligations returns post-issuance obligations for eligible products
func productObligations(productID string, profile CustomerProfile) []string {
	income, _ := profile.Fields["monthly_income"].(float64)
	riskLevel, _ := profile.Fields["risk_level"].(string)

	switch productID {
	case "P001": // personal loan
		return []string{
			"بازپرداخت اقساط ماهانه حداکثر تا روز سررسید",
			fmt.Sprintf("قسط ماهانه نباید از ۴۰٪ درآمد (%s) تجاوز کند", formatNumber(income*0.4)),
			"در صورت تأخیر بیش از ۳۰ روز، جریمه دیرکرد اعمال می‌شود",
			"تغییر شغل یا کاهش درآمد باید ظرف ۱۴ روز اعلام شود",
		}
	case "P002": // housing loan
		return []string{
			"ارائه وثیقه ملکی به ارزش حداقل ۱۲۰٪ مبلغ وام",
			"بیمه عمر و آتش‌سوزی ملک تا پایان دوره بازپرداخت الزامی است",
			"بازپرداخت اقساط حداکثر تا روز سررسید — تأخیر منجر به اجرای وثیقه می‌شود",
			"انتقال ملک تا تسویه کامل وام ممنوع است",
		}
	case "P003": // checkbook
		obs := []string{
			"نگهداری حداقل موجودی ۱۰ میلیون تومان در حساب جاری",
			"ممنوعیت صدور چک بلامحل — در صورت برگشت چک، دسته‌چک مسدود و گزارش به سامانه صیاد ارسال می‌شود",
			"ثبت تمام چک‌ها در سامانه صیاد قبل از تحویل به ذینفع الزامی است",
			"حداکثر تعداد برگ چک در هر دوره: ۲۵ برگ",
		}
		if riskLevel == "medium" {
			obs = append(obs, "به‌دلیل سطح ریسک متوسط: سقف مبلغ هر برگ چک محدود به ۵۰ میلیون تومان است")
		}
		return obs
	case "P004": // credit card
		limit := income * 2
		if riskLevel == "low" {
			limit = income * 3
		}
		return []string{
			fmt.Sprintf("سقف اعتبار: %s", formatNumber(limit)),
			"پرداخت حداقل ۱۰٪ موجودی بدهی تا تاریخ سررسید ماهانه",
			"نرخ کارمزد گردشی: ۲٪ ماهانه بر مانده بدهی",
			"در صورت عدم پرداخت ۳ ماه متوالی، کارت مسدود و به وصول مطالبات ارجاع می‌شود",
		}
	case "P007": // business loan
		return []string{
			"ارائه گزارش عملکرد فصلی کسب‌وکار",
			"وثیقه ملکی یا ضمانت‌نامه بانکی به ارزش ۱۰۰٪ مبلغ وام",
			"استفاده از تسهیلات صرفاً در محل کسب اعلام‌شده",
			"بازپرداخت اقساط حداکثر تا روز سررسید",
		}
	case "P010": // marriage loan
		return []string{
			"ارائه سند ازدواج رسمی حداکثر ۶ ماه پس از دریافت وام",
			"بازپرداخت اقساط ماهانه حداکثر تا روز سررسید",
			"در صورت طلاق قبل از تسویه، مانده بدهی حال می‌شود",
		}
	}
	return nil
}

// productCreditLimit calculates credit/loan limit based on income and risk
func productCreditLimit(productID string, profile CustomerProfile) string {
	income, _ := profile.Fields["monthly_income"].(float64)
	riskLevel, _ := profile.Fields["risk_level"].(string)
	deposits, _ := profile.Fields["total_deposits"].(float64)
	turnover, _ := profile.Fields["account_turnover_12m"].(float64)

	multiplier := 1.0
	switch riskLevel {
	case "low":
		multiplier = 1.5
	case "medium":
		multiplier = 1.0
	case "high":
		multiplier = 0.5
	}

	switch productID {
	case "P001":
		limit := income * 12 * multiplier
		if limit > 500_000_000 {
			limit = 500_000_000
		}
		return fmt.Sprintf("سقف وام شخصی: %s (بر اساس درآمد سالانه × ضریب ریسک)", formatNumber(limit))
	case "P002":
		limit := math.Min(income*48*multiplier, 2_000_000_000)
		return fmt.Sprintf("سقف وام مسکن: %s", formatNumber(limit))
	case "P003":
		perCheck := income * 2 * multiplier
		if riskLevel == "medium" && perCheck > 50_000_000 {
			perCheck = 50_000_000
		}
		return fmt.Sprintf("سقف هر برگ چک: %s | سقف ماهانه: %s", formatNumber(perCheck), formatNumber(perCheck*5))
	case "P004":
		limit := income * 2 * multiplier
		if riskLevel == "low" {
			limit = income * 3
		}
		return fmt.Sprintf("سقف اعتبار کارت: %s", formatNumber(limit))
	case "P007":
		limit := math.Min(turnover*0.3*multiplier, 5_000_000_000)
		return fmt.Sprintf("سقف وام کسب‌وکار: %s (۳۰٪ گردش سالانه)", formatNumber(limit))
	case "P010":
		return "سقف وام ازدواج: طبق بخشنامه بانک مرکزی (مبلغ ثابت مصوب)"
	case "P005", "P006":
		if deposits > 0 {
			return fmt.Sprintf("سپرده‌گذاری پیشنهادی بر اساس موجودی فعلی: %s", formatNumber(deposits))
		}
	}
	return ""
}

// GenerateDefaultWarning produces consequences of payment default
func GenerateDefaultWarning(currentRisk string) *models.DefaultWarning {
	w := &models.DefaultWarning{CurrentRiskLevel: currentRisk}

	switch currentRisk {
	case "low":
		w.PotentialRiskLevel = "medium"
		w.Consequences = []string{
			"Risk level will increase to medium",
			"Checkbook issuance may be suspended",
			"Future loan applications will require additional guarantees",
			"Credit card limit may be reduced",
		}
		w.ConsequencesFa = []string{
			"سطح ریسک از کم به متوسط افزایش می‌یابد",
			"صدور دسته‌چک ممکن است معلق شود",
			"درخواست وام آینده نیازمند ضمانت‌های اضافی خواهد بود",
			"سقف کارت اعتباری ممکن است کاهش یابد",
		}
	case "medium":
		w.PotentialRiskLevel = "high"
		w.Consequences = []string{
			"Risk level will increase to high",
			"All loan applications will be rejected",
			"Checkbook will be revoked",
			"Credit card will be suspended",
			"Legal collection proceedings may be initiated",
		}
		w.ConsequencesFa = []string{
			"سطح ریسک از متوسط به بالا افزایش می‌یابد",
			"تمام درخواست‌های وام رد خواهد شد",
			"دسته‌چک مسدود خواهد شد",
			"کارت اعتباری معلق خواهد شد",
			"اقدامات حقوقی وصول مطالبات ممکن است آغاز شود",
		}
	case "high":
		w.PotentialRiskLevel = "high"
		w.Consequences = []string{
			"Already at highest risk level",
			"All banking services except basic account are restricted",
			"Active legal proceedings for debt collection",
			"Credit bureau negative report will be filed",
		}
		w.ConsequencesFa = []string{
			"در بالاترین سطح ریسک قرار دارید",
			"تمام خدمات بانکی به جز حساب پایه محدود شده است",
			"اقدامات حقوقی وصول فعال است",
			"گزارش منفی به سامانه اعتبارسنجی ارسال خواهد شد",
		}
	}
	return w
}

// ColdStartRisk assesses risk for non-customers based on self-declared info
func ColdStartRisk(req models.ColdStartRequest) models.RiskAssessment {
	score := 50.0 // start at medium

	// Age adjustments
	if req.Age >= 25 && req.Age <= 55 {
		score -= 10
	}
	if req.Age < 21 || req.Age > 65 {
		score += 15
	}

	// Occupation adjustments
	switch req.Occupation {
	case "employee", "manager":
		score -= 15
	case "self_employed":
		score -= 5
	case "retired":
		score -= 10
	case "housewife":
		score += 5
	case "unemployed", "student":
		score += 20
	}

	// Employment type
	switch req.EmploymentType {
	case "government":
		score -= 10
	case "private":
		score -= 5
	}

	// Income adjustments
	if req.ApproxIncome >= 50_000_000 {
		score -= 15
	} else if req.ApproxIncome >= 20_000_000 {
		score -= 5
	} else if req.ApproxIncome < 10_000_000 {
		score += 10
	}

	// Clamp
	score = math.Max(5, math.Min(95, score))

	level := "medium"
	reason := "ارزیابی اولیه بر مبنای اطلاعات خوداظهاری (بدون سابقه بانکی)"
	if score <= 35 {
		level = "low"
		reason = "ریسک کم بر اساس ارزیابی اولیه — نیازمند تأیید با سابقه بانکی"
	} else if score >= 65 {
		level = "high"
		reason = "ریسک بالا بر اساس ارزیابی اولیه — نیازمند بررسی بیشتر"
	}

	return models.RiskAssessment{
		CustomerID: "NEW",
		RiskLevel:  level,
		RiskScore:  score,
		Reason:     reason,
		IsColdStart: true,
	}
}

// --- helpers ---

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func calculateScore(product models.Product, profile CustomerProfile) float64 {
	score := 50.0
	income, _ := profile.Fields["monthly_income"].(float64)
	riskScore, _ := profile.Fields["risk_score"].(float64)

	// Higher income = better fit for loans
	if product.Category == "loan" && income > 30_000_000 {
		score += 20
	}
	// Low risk = bonus
	if riskScore < 30 {
		score += 15
	}
	// Deposits always a safe recommendation
	if product.Category == "deposit" {
		score += 10
	}
	// Cold-start: slight penalty (less certainty)
	if profile.IsColdStart {
		score -= 5
	}
	// Visit purpose boost — PDF: personalized offer should match why they came
	score += visitPurposeBoost(product, profile.VisitPurpose)
	return math.Max(0, math.Min(100, score))
}

// visitPurposeBoost ranks products matching the customer's stated goal higher
func visitPurposeBoost(product models.Product, purpose string) float64 {
	if purpose == "" {
		return 0
	}
	p := strings.ToLower(purpose)
	// Persian + English keywords
	switch {
	case containsAny(p, "وام", "تسهیلات", "loan", "مرابحه", "ازدواج"):
		if product.Category == "loan" {
			// marriage-specific
			if containsAny(p, "ازدواج", "marriage") && product.ID == "P010" {
				return 25
			}
			if containsAny(p, "مسکن", "housing", "خانه") && product.ID == "P002" {
				return 25
			}
			if containsAny(p, "کسب", "business", "تجارت") && product.ID == "P007" {
				return 25
			}
			return 15
		}
	case containsAny(p, "چک", "دسته‌چک", "دسته چک", "check"):
		if product.ID == "P003" {
			return 25
		}
	case containsAny(p, "کارت", "اعتبار", "credit"):
		if product.ID == "P004" {
			return 25
		}
	case containsAny(p, "سپرده", "پس‌انداز", "deposit"):
		if product.Category == "deposit" {
			return 20
		}
	case containsAny(p, "اینترنت", "پیامک", "آنلاین", "بانکداری"):
		if product.Category == "service" {
			return 15
		}
	}
	return 0
}

func containsAny(s string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(s, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

func fieldNameFa(field string) string {
	names := map[string]string{
		"age": "سن", "gender": "جنسیت", "occupation": "شغل",
		"employment_type": "نوع اشتغال", "monthly_income": "درآمد ماهانه",
		"account_turnover_3m": "گردش حساب ۳ ماهه", "account_turnover_12m": "گردش حساب ۱۲ ماهه",
		"total_deposits": "مجموع سپرده‌ها", "active_loans": "وام‌های فعال",
		"installment_default": "اقساط معوق", "payment_history": "سابقه پرداخت",
		"risk_level": "سطح ریسک", "risk_score": "امتیاز ریسک",
		"spending_pattern": "الگوی هزینه", "has_guarantor": "ضامن",
		"customer_type": "نوع مشتری",
	}
	if fa, ok := names[field]; ok {
		return fa
	}
	return field
}

func operatorFa(op string) string {
	ops := map[string]string{
		"gt": "بیشتر از", "gte": "حداقل", "lt": "کمتر از", "lte": "حداکثر",
		"eq": "برابر با", "neq": "نابرابر با",
	}
	if fa, ok := ops[op]; ok {
		return fa
	}
	return op
}

func formatNumber(n float64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1f میلیارد", n/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.0f میلیون", n/1_000_000)
	}
	if n == float64(int(n)) {
		return fmt.Sprintf("%d", int(n))
	}
	return fmt.Sprintf("%.1f", n)
}

// FormatCurrency formats a number to Persian-readable currency
func FormatCurrency(n float64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1f میلیارد تومان", n/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.0f میلیون تومان", n/1_000_000)
	}
	return fmt.Sprintf("%.0f تومان", n)
}

// PersonalizedOffers returns top N offers from eligible products
func PersonalizedOffers(eligible []models.ProductMatch, n int) []models.ProductMatch {
	if n <= 0 {
		return nil
	}
	if len(eligible) <= n {
		return eligible
	}
	// Already sorted by score in MatchAllProducts
	return eligible[:n]
}

// ColdStartNotes returns system notes for non-customer match responses
func ColdStartNotes() []string {
	return []string{
		"این ارزیابی بر مبنای اطلاعات خوداظهاری (Self-declared) انجام شده و فاقد سابقه تراکنش است",
		"افرها و اهلیت اعلام‌شده مشروط به افتتاح حساب و ارائه مدارک تکمیلی است",
		"پس از تشکیل پرونده، ارزیابی ریسک RBCI به‌روزرسانی خواهد شد",
	}
}
