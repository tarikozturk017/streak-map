package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tarikozturk017/streak-map/backend/internal/auth"
	"github.com/tarikozturk017/streak-map/backend/internal/database"
	"github.com/tarikozturk017/streak-map/backend/internal/handlers"
	"github.com/tarikozturk017/streak-map/backend/internal/middleware"
)

func main() {
	db, err := database.NewConnection(
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_NAME", "streakmap"),
	)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := db.AutoMigrate(); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	jwtService := auth.NewJWTService(
		getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
		15*time.Minute, // access token TTL
		7*24*time.Hour, // refresh token TTL
	)

	authHandler := handlers.NewAuthHandler(db.DB, jwtService)
	authMiddleware := middleware.AuthMiddleware(jwtService)

	mux := http.NewServeMux()
	
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.Handle("GET /auth/me", authMiddleware(http.HandlerFunc(authHandler.Me)))

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}