package main

// سرور HTTP بک‌اند دستیار بانکی — پورت پیش‌فرض 8080
// مسیرها: /api/identity, /financial, /rbci, /products, /circulars, /match, /match/cold-start, /health

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/banking-assistant/backend/internal/handlers"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", handlers.HealthHandler)

	// Identity service
	mux.HandleFunc("GET /api/identity", handlers.IdentityHandler)

	// Financial service
	mux.HandleFunc("GET /api/financial", handlers.FinancialHandler)

	// RBCI risk service
	mux.HandleFunc("GET /api/rbci", handlers.RBCIHandler)
	mux.HandleFunc("POST /api/rbci/cold-start", handlers.ColdStartHandler)

	// Products
	mux.HandleFunc("GET /api/products", handlers.ProductsHandler)

	// Circulars
	mux.HandleFunc("GET /api/circulars", handlers.CircularsHandler)
	mux.HandleFunc("GET /api/circulars/by-product", handlers.CircularsByProductHandler)

	// Matching engine
	mux.HandleFunc("POST /api/match", handlers.MatchHandler)
	mux.HandleFunc("POST /api/match/cold-start", handlers.MatchColdStartHandler)

	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8080"
	}

	handler := corsMiddleware(mux)
	fmt.Printf("Banking Assistant Backend running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
