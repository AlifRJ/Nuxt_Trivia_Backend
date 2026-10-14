package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AlifRJ/Go_Auth/app/handler"
	cmiddleware "github.com/AlifRJ/Go_Auth/app/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

func setupRouter(
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	scoreHandler *handler.ScoreHandler,
	leaderboardHandler *handler.LeaderboardHandler,
) chi.Router {
	r := chi.NewRouter()

	// Parse allowed origins from environment variable (comma-separated for multi-domain support)
	rawOrigins := os.Getenv("ALLOWED_ORIGIN")
	allowedOrigins := []string{}
	if rawOrigins != "" {
		allowedOrigins = strings.Split(rawOrigins, ",")
	} else {
		allowedOrigins = []string{"http://localhost:3000"} // Dev fallback
	}

	// CORS Setup
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "Set-Cookie"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Core Middlewares
	r.Use(middleware.RequestID)
	r.Use(cmiddleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Rate Limiting
	r.Use(httprate.LimitBy(5, time.Second, func(r *http.Request) (string, error) {
		return httprate.CanonicalizeIP(r.RemoteAddr), nil
	}))

	// Health Check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Handlers Registry
	authHandler.RegisterRoutes(r)
	userHandler.RegisterRoutes(r)
	scoreHandler.RegisterRoutes(r)
	leaderboardHandler.RegisterRoutes(r)

	return r
}