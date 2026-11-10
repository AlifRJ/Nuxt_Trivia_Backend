package handler

import (
	"net/http"
	"time"

	"github.com/AlifRJ/Go_Auth/app/middleware"
	"github.com/AlifRJ/Go_Auth/app/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

type AnalyticsHandler struct {
	service service.AnalyticsService
}

func NewAnalyticsHandler(s service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: s,
	}
}

// RegisterRoutes registers endpoints for /v1/analytics
func (h *AnalyticsHandler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/analytics", func(r chi.Router) {

		r.Use(middleware.RequireDashboardToken)

		// Read Operations (60 requests/minute)
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitBy(60, time.Minute, GetUserIDRateKey, httprate.WithLimitHandler(middleware.WriteRateLimitError)))

			// GET /v1/analytics - Dashboard metrics for Laravel Inertia
			r.Get("/", h.GetDashboardAnalytics)
		})
	})
}

// GetDashboardAnalytics handles fetching aggregate dashboard stats
func (h *AnalyticsHandler) GetDashboardAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	analytics, err := h.service.GetDashboardAnalytics(ctx)
	if err != nil {
		middleware.WriteJSONError(w, http.StatusInternalServerError, "Failed to load dashboard analytics: "+err.Error())
		return
	}

	middleware.WriteJSONResponse(w, http.StatusOK, analytics)
}
