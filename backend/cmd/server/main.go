package main

// سرور HTTP بک‌اند دستیار بانکی — پورت پیش‌فرض 8080
// مسیرها: /api/identity, /api/financial, /api/rbci, /api/rbci/customers, /api/products, /api/circulars, /api/match, /api/match/cold-start, /api/health

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/banking-assistant/backend/internal/data"
	"github.com/banking-assistant/backend/internal/handlers"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := initPostgresWithRetry(ctx, dsn); err != nil {
			log.Fatalf("postgres init failed: %v", err)
		}
		defer data.ClosePostgres()
		log.Println("local RBCI PostgreSQL endpoint is ready")
	}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", handlers.HealthHandler)

	// Identity service
	mux.HandleFunc("GET /api/identity", handlers.IdentityHandler)
	mux.HandleFunc("/api/customers", handlers.CustomersHandler)
	mux.HandleFunc("/api/customers/{national_id}", handlers.CustomerHandler)
	mux.HandleFunc("/api/rbci/customers", handlers.CustomersHandler)
	mux.HandleFunc("/api/rbci/customers/{national_id}", handlers.CustomerHandler)

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

func initPostgresWithRetry(ctx context.Context, dsn string) error {
	var lastErr error
	for {
		if err := data.InitPostgres(ctx, dsn); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return lastErr
		case <-time.After(time.Second):
		}
	}
}
