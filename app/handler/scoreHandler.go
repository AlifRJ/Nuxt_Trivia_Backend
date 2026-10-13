package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/AlifRJ/Go_Auth/app/middleware"
	"github.com/AlifRJ/Go_Auth/app/model"
	"github.com/AlifRJ/Go_Auth/app/repository"
	"github.com/AlifRJ/Go_Auth/app/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth/v5"
)

type ScoreHandler struct {
	service         *service.ScoreService
	authRepo        model.AuthRepository
	accessTokenAuth *jwtauth.JWTAuth
}

func NewScoreHandler(s *service.ScoreService, authRepo model.AuthRepository, accessTokenAuth *jwtauth.JWTAuth) *ScoreHandler {
	return &ScoreHandler{
		service:         s,
		authRepo:        authRepo,
		accessTokenAuth: accessTokenAuth,
	}
}

// RegisterRoutes sets up all routes and middleware for /v1/scores
func (h *ScoreHandler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/scores", func(r chi.Router) {
		r.Use(jwtauth.Verifier(h.accessTokenAuth))
		r.Use(jwtauth.Authenticator(h.accessTokenAuth))
		r.Use(middleware.CheckTokenBlacklist(h.authRepo))

		// Helper Fetch claim from JWT
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

		// Read Group (60 requests/minute)
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitBy(60, time.Minute, GetUserIDRateKey, httprate.WithLimitHandler(middleware.WriteRateLimitError)))

			r.Get("/", h.GetAll)
			r.Get("/user/{userID}", h.GetByUser)
		})

		// Create Group (15 requests/minute)
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitBy(15, time.Minute, GetUserIDRateKey, httprate.WithLimitHandler(middleware.WriteRateLimitError)))

			r.Post("/", h.Create)
		})

		// Delete Group (3 requests/minute)
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitBy(3, time.Minute, GetUserIDRateKey, httprate.WithLimitHandler(middleware.WriteRateLimitError)))

			r.Delete("/{id}", h.DeleteScore)
			r.Delete("/{id}/permanent", h.PermanentlyDeleteScore)
		})
	})
}

// GetAll fetches paginated scores
func (h *ScoreHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))

	scores, err := h.service.GetAll(r.Context(), limit, offset)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch scores"})
		return
	}

	respondJSON(w, http.StatusOK, scores)
}

// GetByUser fetches all scores created by a specific user
func (h *ScoreHandler) GetByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUintParam(r, "userID")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid userID parameter"})
		return
	}

	scores, err := h.service.GetByUser(r.Context(), userID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch user scores"})
		return
	}

	respondJSON(w, http.StatusOK, scores)
}

// Create registers a new score record
func (h *ScoreHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID     uint   `json:"user_id"`
		Score      uint   `json:"score"`
		Category   string `json:"category"`
		Difficulty string `json:"difficulty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		return
	}

	score, err := h.service.RegisterScore(r.Context(), body.UserID, body.Score, body.Category, body.Difficulty)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create score"})
		return
	}

	respondJSON(w, http.StatusCreated, score)
}

// DeleteScore performs a soft delete on a score record
func (h *ScoreHandler) DeleteScore(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID parameter"})
		return
	}

	if err := h.service.DeleteScore(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrScoreNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Score not found"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PermanentlyDeleteScore hard deletes a score record from the database
func (h *ScoreHandler) PermanentlyDeleteScore(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID parameter"})
		return
	}

	if err := h.service.DeleteScorePermanently(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrScoreNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Score not found"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}