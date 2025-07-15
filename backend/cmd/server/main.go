import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-change-this-in-production")
	if jwtSecret == "your-secret-key-change-this-in-production" {
		log.Fatal("CRITICAL: Default JWT_SECRET is used. This is insecure. Please set a strong secret for production.")
	}
	jwtService := auth.NewJWTService(
		jwtSecret,
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
	
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}