package server

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"fleet-tracker-service/internal/auth"
	"fleet-tracker-service/internal/handlers"
	"fleet-tracker-service/internal/repository"
	"fleet-tracker-service/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func Run() error {
	_ = godotenv.Load()

	port := mustGetenv("PORT", "8080")
	jwtSecret := mustGetenv("JWT_SECRET", "fleet-tracker")
	pgDSN := mustGetenv("DB_URL", "postgres://postgres:postgres@postgres:5432/fleet?sslmode=disable")
	redisAddr := mustGetenv("REDIS_ADDR", "redis:6379")

	// Run migrations
	if err := runMigrations(pgDSN); err != nil {
		return err
	}

	// Initialize Postgres DB
	db, err := sql.Open("postgres", pgDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return err
	}

	// Initialize Redis DB
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return err
	}

	// To Setup dependencies
	repo := repository.NewRepo(db)
	svc := service.NewService(repo, rdb)
	authSvc := auth.NewJWT([]byte(jwtSecret))

	// To Setup Gin Router
	router := gin.New()
	router.Use(gin.Recovery(), auth.RequestLogger())

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Authorization", "Content-Type"},
	}))

	// To redirect to Swagger UI by default
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/docs/swagger/index.html")
	})

	router.POST("/login", handlers.LoginHandler(authSvc))

	// To Serve static Swagger folder
	router.Static("/docs/swagger", "./docs/swagger")

	// To Protected API routes
	api := router.Group("/api")
	api.Use(auth.JWTMiddleware(authSvc))
	{
		api.POST("/vehicle/ingest", handlers.IngestHandler(svc))
		api.GET("/vehicle/status", handlers.StatusHandler(svc))
		api.GET("/vehicle/trips", handlers.TripsHandler(svc))
	}

	// Start simulator in background
	go svc.StartSimulator(context.Background())

	// Start server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("Server listening on %s", srv.Addr)
		log.Printf("Swagger UI available at http://localhost:%s/docs/swagger/index.html", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("Server exited")
	return nil
}

func mustGetenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
