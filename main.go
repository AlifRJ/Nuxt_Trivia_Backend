package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlifRJ/Go_Auth/app/handler"
	"github.com/AlifRJ/Go_Auth/app/migration"
	"github.com/AlifRJ/Go_Auth/app/repository"
	"github.com/AlifRJ/Go_Auth/app/service"
	"github.com/go-chi/jwtauth/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found, relying on system env")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Databases setup (from config.go)
	db := initPostgres(ctx)
	defer db.Close()

	rdb := initRedis(ctx)
	defer rdb.Close()

	// Run migrations
	if err := migration.UserMigration(ctx, db); err != nil {
		log.Fatalf("Failed to execute user migration: %v\n", err)
	}
	if err := migration.AuthMigration(ctx, db); err != nil {
		log.Fatalf("Failed to execute user migration: %v\n", err)
	}
	if err := migration.ScoreMigration(ctx, db); err != nil {
		log.Fatalf("Failed to execute score migration: %v\n", err)
	}

	// JWT Setup
	accessSecret := os.Getenv("ACCESS_SECRET_KEY")
	refreshSecret := os.Getenv("REFRESH_SECRET_KEY")
	if accessSecret == "" || refreshSecret == "" {
		log.Fatal("JWT Secret Keys must be configured in environment variables")
	}
	accessTokenAuth := jwtauth.New("HS256", []byte(accessSecret), nil)
	refreshTokenAuth := jwtauth.New("HS256", []byte(refreshSecret), nil)

	// Dependency Injection
	userRepo := repository.NewPostgresUserRepository(db)
	scoreRepo := repository.NewPostgresScoreRepository(db)
	authRepo := repository.NewPostgresAuthRepository(db)

	cachedUserRepo := repository.NewCachedUserRepository(rdb, userRepo, 15*time.Minute)
	cachedScoreRepo := repository.NewCachedScoreRepository(rdb, scoreRepo, 15*time.Minute)
	cachedAuthRepo := repository.NewCachedAuthRepository(rdb, authRepo)

	userService := service.NewUserService(cachedUserRepo, cachedAuthRepo)
	leaderboardService := service.NewLeaderboardService(rdb)
	scoreService := service.NewScoreService(cachedScoreRepo, leaderboardService)
	authService := service.NewAuthService(cachedUserRepo, cachedAuthRepo, accessTokenAuth, refreshTokenAuth)

	userHandler := handler.NewUserHandler(userService, cachedAuthRepo, accessTokenAuth)
	scoreHandler := handler.NewScoreHandler(scoreService, cachedAuthRepo, accessTokenAuth)
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardService, cachedAuthRepo, accessTokenAuth)
	authHandler := handler.NewAuthHandler(authService, userService, cachedAuthRepo, accessTokenAuth, refreshTokenAuth)

	// Router setup (from router.go)
	router := setupRouter(authHandler, userHandler, scoreHandler, leaderboardHandler)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Printf("Server starting on port: %s\n", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("\nShutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server stopped cleanly.")
}