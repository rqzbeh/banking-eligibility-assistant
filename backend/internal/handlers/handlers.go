// Package handlers — لایه HTTP بک‌اند دستیار بانکی
//
// هر handler یک Mock API یا موتور تطبیق را در دسترس قرار می‌دهد.
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
	profile, ok := data.Identities[nid]
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
	profile, ok := data.Financials[cid]
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
	risk, ok := data.Risks[cid]
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

	identity, idOk := data.Identities[req.NationalID]
	if !idOk {
		writeError(w, 404, "Customer not found — use /api/match/cold-start for non-customers",
			"مشتری یافت نشد — برای افراد غیرمشتری از مسیر cold-start استفاده کنید")
		return
	}

	// مدیریت خطای سامانه‌های بالادستی (الزام PDF)
	var upstreamErrs []string
	financial, finOk := data.Financials[identity.CustomerID]
	if !finOk {
		upstreamErrs = append(upstreamErrs, "سرویس اطلاعات مالی در دسترس نیست")
	}
	risk, riskOk := data.Risks[identity.CustomerID]
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
	writeJSON(w, 200, map[string]string{"status": "ok", "service": "banking-assistant-backend"})
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
