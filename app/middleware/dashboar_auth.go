package middleware

import (
	"net/http"
	"os"
)

// RequireDashboardToken validates an internal static API key header sent by Laravel Guzzle
func RequireDashboardToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Dashboard-Token")
		expectedToken := os.Getenv("DASHBOARD_API_KEY")

		if token == "" || token != expectedToken {
			WriteJSONError(w, http.StatusUnauthorized, "Unauthorized dashboard client")
			return
		}

		next.ServeHTTP(w, r)
	})
}
