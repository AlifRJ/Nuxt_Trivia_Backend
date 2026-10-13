package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/AlifRJ/Go_Auth/app/middleware"
	"github.com/AlifRJ/Go_Auth/app/model"
	"github.com/AlifRJ/Go_Auth/app/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth/v5"
)

type LeaderboardHandler struct {
	service         *service.LeaderboardService
	authRepo        model.AuthRepository
	accessTokenAuth *jwtauth.JWTAuth
}

func NewLeaderboardHandler(s *service.LeaderboardService, authRepo model.AuthRepository, accessTokenAuth *jwtauth.JWTAuth) *LeaderboardHandler {
	return &LeaderboardHandler{
		service:         s,
		authRepo:        authRepo,
		accessTokenAuth: accessTokenAuth,
	}
}

// RegisterRoutes registers endpoints for /v1/leaderboards
func (h *LeaderboardHandler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/leaderboards", func(r chi.Router) {
		r.Use(jwtauth.Verifier(h.accessTokenAuth))
		r.Use(jwtauth.Authenticator(h.accessTokenAuth))
		r.Use(middleware.CheckTokenBlacklist(h.authRepo))

		// getUserIDKey := func(r *http.Request) (string, error) {
		// 	_, claims, err := jwtauth.FromContext(r.Context())
		// 	if err != nil {
		// 		return httprate.CanonicalizeIP(r.RemoteAddr), nil
		// 	}
		// 	if userID, ok := claims["user_id"].(string); ok && userID != "" {
		// 		return userID, nil
		// 	}
		// 	return httprate.CanonicalizeIP(r.RemoteAddr), nil
		// }

		// Read Operations (60 requests/minute)
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitBy(60, time.Minute, GetUserIDRateKey, httprate.WithLimitHandler(middleware.WriteRateLimitError)))

			// GET /v1/leaderboards/{category}?limit=10
			r.Get("/{category}", h.GetTopScores)

			// GET /v1/leaderboards/{category}/user/{userID}
			r.Get("/{category}/user/{userID}", h.GetUserRank)
		})
	})
}

// GetTopScores fetches the top N players for a given category (default limit: 10)
func (h *LeaderboardHandler) GetTopScores(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	if category == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Category parameter is required"})
		return
	}

	limitParam := r.URL.Query().Get("limit")
	limit := 10 // Default limit
	if limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Cap max limit to 100 entries per request
	if limit > 100 {
		limit = 100
	}

	scores, err := h.service.GetTopScores(r.Context(), category, int64(limit))
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch leaderboard"})
		return
	}

	respondJSON(w, http.StatusOK, scores)
}

// GetUserRank retrieves rank and score position for a specific user
func (h *LeaderboardHandler) GetUserRank(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	if category == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Category parameter is required"})
		return
	}

	userID, err := parseUintParam(r, "userID")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid userID parameter"})
		return
	}

	userRank, err := h.service.GetUserRank(r.Context(), category, userID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch user rank"})
		return
	}

	if userRank == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "User not found in leaderboard"})
		return
	}

	respondJSON(w, http.StatusOK, userRank)
}