package middleware

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AlifRJ/Go_Auth/app/model"
	"github.com/go-chi/jwtauth/v5"
)

// CheckTokenBlacklist checks if the JTI claim in the JWT context has been revoked
func CheckTokenBlacklist(authRepo model.AuthRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract claims injected by jwtauth.Authenticator / Verifier
			_, claims, err := jwtauth.FromContext(r.Context())
			if err != nil {
				respondJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized: invalid context token",
				})
				return
			}

			// Extract JTI claim
			jti, ok := claims["jti"].(string)
			if !ok || jti == "" {
				respondJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized: missing token identifier (jti)",
				})
				return
			}

			// Check Blacklist Status via Redis / AuthRepo
			blacklisted, err := authRepo.IsTokenBlacklisted(r.Context(), jti)
			if err != nil {
				// Fail-Open: Log warning for observability, but do NOT block valid JWT requests
				log.Printf("[WARN] Blacklist check fallback (Redis unreachable) for JTI %s: %v", jti, err)
				blacklisted = false
			}

			if blacklisted {
				respondJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "Token has been revoked. Please log in again.",
				})
				return
			}

			// Pass to next middleware/handler
			next.ServeHTTP(w, r)
		})
	}
}

// JSON Response Helper 
func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}