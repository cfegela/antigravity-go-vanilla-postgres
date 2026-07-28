package main

import (
	"log"
	"net/http"
	"os"

	"go-vanilla-crud/db"
	"go-vanilla-crud/handlers"
	"go-vanilla-crud/middleware"
)

// corsMiddleware adds proper CORS headers allowing credentials for HttpOnly cookies
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	log.Println("Initializing Go Task Manager Backend API...")

	// Initialize Database
	db.InitDB()

	mux := http.NewServeMux()

	// Public Auth Endpoints
	mux.HandleFunc("POST /api/auth/register", handlers.RegisterHandler)
	mux.HandleFunc("POST /api/auth/login", handlers.LoginHandler)
	mux.HandleFunc("POST /api/auth/refresh", handlers.RefreshTokenHandler)
	mux.HandleFunc("POST /api/auth/logout", handlers.LogoutHandler)

	// Protected Auth Endpoints
	mux.HandleFunc("GET /api/auth/me", middleware.AuthMiddleware(handlers.MeHandler))

	// Protected CRUD Tasks Endpoints
	mux.HandleFunc("/api/tasks", middleware.AuthMiddleware(handlers.TaskRouter))
	mux.HandleFunc("/api/tasks/", middleware.AuthMiddleware(handlers.TaskRouter))

	// Healthcheck Endpoint
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"go-backend"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler := corsMiddleware(mux)

	log.Printf("Go Backend API Server listening on port %s...", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
