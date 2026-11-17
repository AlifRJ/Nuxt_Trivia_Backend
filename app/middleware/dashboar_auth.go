package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
)

// RequireDashboardToken validates an internal static API key header sent by Laravel Guzzle
func RequireDashboardToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedToken := os.Getenv("DASHBOARD_API_KEY")
		if expectedToken == "" {
			WriteJSONError(w, http.StatusInternalServerError, "Dashboard API key not configured")
			return
		}

		token := r.Header.Get("X-Dashboard-Token")
		if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			WriteJSONError(w, http.StatusUnauthorized, "Unauthorized dashboard client")
			return
		}

		next.ServeHTTP(w, r)
	})
}
