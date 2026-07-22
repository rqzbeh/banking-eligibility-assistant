// Package handlers — لایه HTTP بک‌اند دستیار بانکی
//
// هر handler یک سرویس محلی RBCI یا موتور تطبیق را در دسترس قرار می‌دهد.
// خطاها همیشه با فیلدهای error (انگلیسی) و error_fa (فارسی) برگردانده می‌شوند.
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/banking-assistant/backend/internal/data"
	"github.com/banking-assistant/backend/internal/engine"
	"github.com/banking-assistant/backend/internal/models"
)

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string, msgFa string) {
	writeJSON(w, code, map[string]string{"error": msg, "error_fa": msgFa})
}

// GET /api/identity?national_id=XXX
// سرویس اطلاعات هویتی — سن، جنسیت، شغل، وضعیت اشتغال، نوع مشتری، تاریخ افتتاح
func IdentityHandler(w http.ResponseWriter, r *http.Request) {
	nid := strings.TrimSpace(r.URL.Query().Get("national_id"))
	if ok, msg := engine.ValidateNationalID(nid); !ok {
		writeError(w, 400, "invalid national_id", msg)
		return
	}
	profile, ok, err := data.GetIdentity(r.Context(), nid)
	if err != nil {
		writeError(w, 500, "database error", "خطا در پایگاه داده")
		return
	}
	if !ok {
		// PDF: تشخیص مشتری غیرموجود → مسیر مکالمه‌ای
		writeError(w, 404, "Customer not found", "مشتری یافت نشد — مشتری غیرموجود است")
		return
	}
	writeJSON(w, 200, profile)
}

// GET /api/financial?customer_id=XXX
// سرویس اطلاعات مالی — گردش، الگوی هزینه، درآمد، سابقه پرداخت
func FinancialHandler(w http.ResponseWriter, r *http.Request) {
	cid := strings.TrimSpace(r.URL.Query().Get("customer_id"))
	if cid == "" {
		writeError(w, 400, "customer_id is required", "شناسه مشتری الزامی است")
		return
	}
	profile, ok, err := data.GetFinancial(r.Context(), cid)
	if err != nil {
		writeError(w, 500, "database error", "خطا در پایگاه داده")
		return
	}
	if !ok {
		writeError(w, 404, "Financial profile not found", "پروفایل مالی یافت نشد — سامانه مالی داده برنگرداند")
		return
	}
	writeJSON(w, 200, profile)
}

// GET /api/rbci?customer_id=XXX
// سامانه RBCI — سطح ریسک، امتیاز عددی، دلیل
func RBCIHandler(w http.ResponseWriter, r *http.Request) {
	cid := strings.TrimSpace(r.URL.Query().Get("customer_id"))
	if cid == "" {
		writeError(w, 400, "customer_id is required", "شناسه مشتری الزامی است")
		return
	}
	risk, ok, err := data.GetRisk(r.Context(), cid)
	if err != nil {
		writeError(w, 500, "database error", "خطا در پایگاه داده")
		return
	}
	if !ok {
		writeError(w, 404, "Risk assessment not found", "ارزیابی ریسک یافت نشد — سامانه RBCI داده برنگرداند")
		return
	}
	writeJSON(w, 200, risk)
}

// POST /api/rbci/cold-start
// ارزیابی ریسک اولیه برای غیرمشتری بر مبنای خوداظهاری (Self-declared)
func ColdStartHandler(w http.ResponseWriter, r *http.Request) {
	var req models.ColdStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body", "بدنه درخواست نامعتبر است")
		return
	}
	if errFa := validateColdStart(req); errFa != "" {
		writeError(w, 400, "validation failed", errFa)
		return
	}
	req.Occupation = engine.NormalizeOccupation(req.Occupation)
	risk := engine.ColdStartRisk(req)
	writeJSON(w, 200, risk)
}

// GET /api/products
func ProductsHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, data.Products)
}

// GET /api/circulars
func CircularsHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, data.Circulars)
}

// GET /api/circulars/by-product?product_id=XXX
func CircularsByProductHandler(w http.ResponseWriter, r *http.Request) {
	pid := strings.TrimSpace(r.URL.Query().Get("product_id"))
	if pid == "" {
		writeError(w, 400, "product_id is required", "شناسه محصول الزامی است")
		return
	}
	byProduct := data.CircularsByProduct()
	rules, ok := byProduct[pid]
	if !ok {
		writeJSON(w, 200, []models.CircularRule{})
		return
	}
	writeJSON(w, 200, rules)
}

// POST /api/match
// تطبیق کامل مشتری موجود: افر شخصی‌سازی‌شده + اهلیت + gap + تعهدات
func MatchHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NationalID            string `json:"national_id"`
		IncludeDefaultWarning bool   `json:"include_default_warning"`
		VisitPurpose          string `json:"visit_purpose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request", "درخواست نامعتبر")
		return
	}
	if ok, msg := engine.ValidateNationalID(req.NationalID); !ok {
		writeError(w, 400, "invalid national_id", msg)
		return
	}

	identity, idOk, err := data.GetIdentity(r.Context(), req.NationalID)
	if err != nil {
		writeError(w, 500, "database error", "خطا در پایگاه داده")
		return
	}
	if !idOk {
		writeError(w, 404, "Customer not found — use /api/match/cold-start for non-customers",
			"مشتری یافت نشد — برای افراد غیرمشتری از مسیر cold-start استفاده کنید")
		return
	}

	// مدیریت خطای سامانه‌های بالادستی (الزام PDF)
	var upstreamErrs []string
	financial, finOk, err := data.GetFinancial(r.Context(), identity.CustomerID)
	if err != nil {
		writeError(w, 500, "database error", "خطا در پایگاه داده")
		return
	}
	if !finOk {
		upstreamErrs = append(upstreamErrs, "سرویس اطلاعات مالی در دسترس نیست")
	}
	risk, riskOk, err := data.GetRisk(r.Context(), identity.CustomerID)
	if err != nil {
		writeError(w, 500, "database error", "خطا در پایگاه داده")
		return
	}
	if !riskOk {
		upstreamErrs = append(upstreamErrs, "سامانه RBCI در دسترس نیست")
	}
	if !finOk || !riskOk {
		writeJSON(w, 503, map[string]interface{}{
			"error":           "upstream_unavailable",
			"error_fa":        "یک یا چند سامانه بالادستی در دسترس نیست",
			"upstream_errors": upstreamErrs,
		})
		return
	}

	profile := engine.BuildProfile(&identity, &financial, &risk)
	profile.VisitPurpose = req.VisitPurpose
	eligible, ineligible := engine.MatchAllProducts(profile)
	offers := engine.PersonalizedOffers(eligible, 3)

	resp := models.MatchResponse{
		CustomerID:         identity.CustomerID,
		NationalID:         identity.NationalID,
		CustomerName:       identity.Name,
		IsExisting:         true,
		IsColdStart:        false,
		RiskLevel:          risk.RiskLevel,
		RiskScore:          risk.RiskScore,
		RiskReason:         risk.Reason,
		VisitPurpose:       req.VisitPurpose,
		EligibleProducts:   eligible,
		IneligibleProducts: ineligible,
		PersonalizedOffers: offers,
	}

	if req.IncludeDefaultWarning {
		resp.DefaultWarning = engine.GenerateDefaultWarning(risk.RiskLevel)
	}

	writeJSON(w, 200, resp)
}

// POST /api/match/cold-start
// مسیر دوم PDF: مشتری غیرموجود — ارزیابی خوداظهاری + افر مشروط
func MatchColdStartHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		models.ColdStartRequest
		IncludeDefaultWarning bool `json:"include_default_warning"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request", "درخواست نامعتبر")
		return
	}
	if errFa := validateColdStart(req.ColdStartRequest); errFa != "" {
		writeError(w, 400, "validation failed", errFa)
		return
	}

	req.Occupation = engine.NormalizeOccupation(req.Occupation)
	risk := engine.ColdStartRisk(req.ColdStartRequest)
	profile := engine.BuildProfileFromColdStart(req.ColdStartRequest, &risk)
	eligible, ineligible := engine.MatchAllProducts(profile)
	offers := engine.PersonalizedOffers(eligible, 3)

	resp := models.MatchResponse{
		CustomerID:         "NEW",
		NationalID:         "",
		CustomerName:       req.Name,
		IsExisting:         false,
		IsColdStart:        true,
		RiskLevel:          risk.RiskLevel,
		RiskScore:          risk.RiskScore,
		RiskReason:         risk.Reason,
		VisitPurpose:       req.VisitPurpose,
		EligibleProducts:   eligible,
		IneligibleProducts: ineligible,
		PersonalizedOffers: offers,
		NotesFa:            engine.ColdStartNotes(),
	}

	if req.IncludeDefaultWarning {
		resp.DefaultWarning = engine.GenerateDefaultWarning(risk.RiskLevel)
	}

	writeJSON(w, 200, resp)
}

// GET /api/health
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok", "service": "banking-assistant-backend", "customer_store": data.CustomerStoreName()})
}

// CustomersHandler exposes the configured local RBCI endpoint.
func CustomersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		customers, err := data.ListCustomers(r.Context())
		if err != nil {
			writeError(w, 500, "database error", "خطا در پایگاه داده")
			return
		}
		writeJSON(w, 200, customers)
	case http.MethodPost:
		var rec models.CustomerRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			writeError(w, 400, "Invalid request", "درخواست نامعتبر")
			return
		}
		if errFa := validateCustomerRecord(&rec); errFa != "" {
			writeError(w, 400, "validation failed", errFa)
			return
		}
		if err := data.CreateCustomer(r.Context(), rec); err != nil {
			if strings.Contains(err.Error(), "customer_id already exists") {
				writeError(w, 409, "customer_id already exists", "شناسه مشتری قبلاً ثبت شده است")
				return
			}
			if strings.Contains(err.Error(), "already exists") {
				writeError(w, 409, "customer already exists", "مشتری با این کد ملی قبلاً ثبت شده است")
				return
			}
			writeError(w, 500, "database error", "خطا در ذخیره مشتری")
			return
		}
		writeJSON(w, 201, rec)
	default:
		writeError(w, 405, "method not allowed", "متد مجاز نیست")
	}
}

func CustomerHandler(w http.ResponseWriter, r *http.Request) {
	nid := strings.TrimSpace(r.PathValue("national_id"))
	if ok, msg := engine.ValidateNationalID(nid); !ok {
		writeError(w, 400, "invalid national_id", msg)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rec, ok, err := data.GetCustomer(r.Context(), nid)
		if err != nil {
			writeError(w, 500, "database error", "خطا در پایگاه داده")
			return
		}
		if !ok {
			writeError(w, 404, "Customer not found", "مشتری یافت نشد")
			return
		}
		writeJSON(w, 200, rec)
	case http.MethodPut:
		var rec models.CustomerRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			writeError(w, 400, "Invalid request", "درخواست نامعتبر")
			return
		}
		rec.Identity.NationalID = strings.TrimSpace(rec.Identity.NationalID)
		if rec.Identity.NationalID == "" {
			rec.Identity.NationalID = nid
		}
		if errFa := validateCustomerRecord(&rec); errFa != "" {
			writeError(w, 400, "validation failed", errFa)
			return
		}
		if err := data.UpdateCustomer(r.Context(), nid, rec); err != nil {
			if strings.Contains(err.Error(), "mismatch") {
				writeError(w, 400, "identity mismatch", "کد ملی یا شناسه مشتری با رکورد موجود یکسان نیست")
				return
			}
			if strings.Contains(err.Error(), "not found") {
				writeError(w, 404, "Customer not found", "مشتری یافت نشد")
				return
			}
			writeError(w, 500, "database error", "خطا در ذخیره مشتری")
			return
		}
		writeJSON(w, 200, rec)
	case http.MethodDelete:
		ok, err := data.DeleteCustomer(r.Context(), nid)
		if err != nil {
			writeError(w, 500, "database error", "خطا در حذف مشتری")
			return
		}
		if !ok {
			writeError(w, 404, "Customer not found", "مشتری یافت نشد")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "deleted"})
	default:
		writeError(w, 405, "method not allowed", "متد مجاز نیست")
	}
}

// --- validation helpers ---

func validateColdStart(req models.ColdStartRequest) string {
	if req.Name == "" {
		return "نام الزامی است"
	}
	if req.Age < 15 || req.Age > 100 {
		return "سن باید بین ۱۵ تا ۱۰۰ باشد"
	}
	if req.Occupation == "" {
		return "شغل الزامی است"
	}
	if req.Gender != "" && req.Gender != "male" && req.Gender != "female" {
		return "جنسیت باید male یا female باشد"
	}
	if req.ApproxIncome < 0 {
		return "درآمد نمی‌تواند منفی باشد"
	}
	return ""
}

func validateCustomerRecord(rec *models.CustomerRecord) string {
	rec.Identity.NationalID = strings.TrimSpace(rec.Identity.NationalID)
	rec.Identity.CustomerID = strings.TrimSpace(rec.Identity.CustomerID)
	rec.Identity.Name = strings.TrimSpace(rec.Identity.Name)
	rec.Identity.Occupation = engine.NormalizeOccupation(rec.Identity.Occupation)
	if ok, msg := engine.ValidateNationalID(rec.Identity.NationalID); !ok {
		return msg
	}
	if rec.Identity.CustomerID == "" {
		return "شناسه مشتری الزامی است"
	}
	if rec.Identity.Name == "" {
		return "نام الزامی است"
	}
	if rec.Identity.Age < 15 || rec.Identity.Age > 100 {
		return "سن باید بین ۱۵ تا ۱۰۰ باشد"
	}
	if rec.Identity.Gender != "" && rec.Identity.Gender != "male" && rec.Identity.Gender != "female" {
		return "جنسیت باید male یا female باشد"
	}
	if rec.Risk.RiskLevel != "low" && rec.Risk.RiskLevel != "medium" && rec.Risk.RiskLevel != "high" {
		return "سطح ریسک باید low، medium یا high باشد"
	}
	if rec.Risk.RiskScore < 0 || rec.Risk.RiskScore > 100 {
		return "امتیاز ریسک باید بین ۰ تا ۱۰۰ باشد"
	}
	if rec.Financial.MonthlyIncome < 0 || rec.Financial.AccountTurnover3M < 0 ||
		rec.Financial.AccountTurnover12M < 0 || rec.Financial.TotalDeposits < 0 ||
		rec.Financial.ActiveLoans < 0 || rec.Financial.TotalLoanAmount < 0 ||
		rec.Financial.InstallmentDefault < 0 {
		return "مقادیر مالی نمی‌توانند منفی باشند"
	}
	rec.Identity.IsExisting = true
	rec.Financial.CustomerID = rec.Identity.CustomerID
	rec.Risk.CustomerID = rec.Identity.CustomerID
	return ""
}
