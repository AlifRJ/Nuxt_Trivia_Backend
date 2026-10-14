package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/AlifRJ/Go_Auth/app/middleware"
	"github.com/AlifRJ/Go_Auth/app/model"
	"github.com/AlifRJ/Go_Auth/app/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth/v5"
)

type AuthHandler struct {
	service 			*service.AuthService
	userService        	*service.UserService
	authRepo        	model.AuthRepository
	accessTokenAuth 	*jwtauth.JWTAuth
	refreshTokenAuth 	*jwtauth.JWTAuth
}

func NewAuthHandler(s *service.AuthService, userService *service.UserService, authRepo model.AuthRepository, accessAuth *jwtauth.JWTAuth, refreshAuth *jwtauth.JWTAuth) *AuthHandler {
	return &AuthHandler{
		service: 			s, 
		userService: 		userService, 
		authRepo:        	authRepo,
		accessTokenAuth: 	accessAuth,
		refreshTokenAuth: 	refreshAuth,
	}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Route("/v1", func(r chi.Router) {

		r.Options("/*", func(w http.ResponseWriter, r *http.Request) {
        	w.WriteHeader(http.StatusOK)
    	})

		// Public Group
		r.Group(func(r chi.Router){
			// Limit public /v1 to 5 request per second per ip address
			r.Use(httprate.LimitBy(5, time.Second, func(r *http.Request)(string, error){
				return httprate.CanonicalizeIP(r.RemoteAddr), nil
			}, httprate.WithLimitHandler(middleware.WriteRateLimitError)))
	
			r.Post("/login", h.Login) 
			r.Post("/register", h.Register) 
		})

		// Logout Group
		r.Group(func(r chi.Router){
			r.Use(jwtauth.Verifier(h.accessTokenAuth))
			r.Use(jwtauth.Authenticator(h.accessTokenAuth))
			r.Use(middleware.CheckTokenBlacklist(h.authRepo))
			
			// JTI Rate limit
			r.Use(httprate.LimitBy(5, time.Minute, func(r *http.Request) (string, error) {
				// Fetch token and claims from context
				_, claims, err := jwtauth.FromContext(r.Context())
				if err != nil {
					return httprate.CanonicalizeIP(r.RemoteAddr), nil
				}

				// Fetch claim Subject
				if userID, ok := claims["user_id"].(string); ok && userID != "" {
					return userID, nil
				}

				return httprate.CanonicalizeIP(r.RemoteAddr), nil
			}, httprate.WithLimitHandler(middleware.WriteRateLimitError)))	

			r.Post("/logout", h.Logout)
		})

		// Refresh Group
		r.Group(func(r chi.Router) {
			r.Use(jwtauth.Verifier(h.refreshTokenAuth))
			r.Use(jwtauth.Authenticator(h.refreshTokenAuth))
			
			// IP rate limit
			r.Use(httprate.LimitBy(3, 10*time.Second, func(r *http.Request) (string, error) {
				return httprate.CanonicalizeIP(r.RemoteAddr), nil
			}))

			// JTI Rate limit
			r.Use(httprate.LimitBy(5, time.Minute, func(r *http.Request) (string, error) {
				// Fetch token and claims from context
				_, claims, err := jwtauth.FromContext(r.Context())
				if err != nil {
					return httprate.CanonicalizeIP(r.RemoteAddr), nil
				}

				// Fetch claim Subject
				if jti, ok := claims["jti"].(string); ok && jti != "" {
					return jti, nil
				}

				return httprate.CanonicalizeIP(r.RemoteAddr), nil
			}, httprate.WithLimitHandler(middleware.WriteRateLimitError)))	
			
			r.Post("/refresh", h.Refresh)
		})

		// Profile Group
		r.Group(func(r chi.Router){
			r.Use(jwtauth.Verifier(h.accessTokenAuth))
			r.Use(jwtauth.Authenticator(h.accessTokenAuth))
			r.Use(middleware.CheckTokenBlacklist(h.authRepo))
			
			// JTI Rate limit
			r.Use(httprate.LimitBy(60, time.Minute, func(r *http.Request) (string, error) {
				// Fetch token and claims from context
				_, claims, err := jwtauth.FromContext(r.Context())
				if err != nil {
					return httprate.CanonicalizeIP(r.RemoteAddr), nil
				}

				// Fetch claim Subject
				if userID, ok := claims["user_id"].(string); ok && userID != "" {
					return userID, nil
				}

				return httprate.CanonicalizeIP(r.RemoteAddr), nil
			}, httprate.WithLimitHandler(middleware.WriteRateLimitError)))	

			r.Get("/me", h.Me)
		})
		
	})
}

// Login Handler
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request){
	defer r.Body.Close()

	// Request Validation
	var body struct {
		Identity  string `json:"identity"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if body.Identity == "" || body.Password == "" {
        respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Identity and password are required"})
        return
    }

	// Get the user
	accessToken, refreshToken, user, err := h.service.Login(r.Context(), body.Identity, body.Password)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	// Return token string to the client
	setRefreshTokenCookie(w, refreshToken, 7*24*time.Hour)

	respondJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"user":         user,
	})
}

// Register Handler
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request){
	// Request Validation
	var body struct {
		Name  		string `json:"name"`
		Username  	string `json:"username"`
		Email 		string `json:"email"`
		Password 	string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		return
	}

	_, err := h.userService.RegisterUser(r.Context(), body.Name, body.Username, body.Email, body.Password)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{
		"message": "User registered successfully. Please login.",
	})
}

// Refresh Handler
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request){
	// Extract the claims injected into the context by the Verifier middleware
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token context"})
		return
	}

	userID, err := extractUserIDFromClaims(claims)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid user ID in claims"})
		return
	}

	// Fetch refresh token
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Refresh token cookie missing"})
		return
	}

	// Rotate token
	newAccessToken, newRefreshToken, err := h.service.RefreshToken(r.Context(), userID, cookie.Value)
	if err != nil {
		clearRefreshTokenCookie(w)
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	// Update refresh token
	setRefreshTokenCookie(w, newRefreshToken, 7*24*time.Hour)

	respondJSON(w, http.StatusOK, map[string]string{
		"access_token": newAccessToken,
	})
}

// User Profile Handler
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"user_claims": claims,
	})
}

// Logout Handler
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	userID, err := extractUserIDFromClaims(claims)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid claims"})
		return
	}

	// Calculate Expiration
	var jti string
	if val, ok := claims["jti"].(string); ok {
		jti = val
	}

	var accessTTL time.Duration
	if token != nil {
		if exp, ok := token.Expiration(); ok {
			accessTTL = time.Until(exp)
		}
	}

	// Invalidate Session & Blacklist JTI
	if err := h.service.Logout(r.Context(), userID, jti, accessTTL); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to logout"})
		return
	}

	clearRefreshTokenCookie(w)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// --- Helper Functions ---

func setRefreshTokenCookie(w http.ResponseWriter, token string, duration time.Duration) {
	isProduction := os.Getenv("APP_ENV") == "production"
	cookieDomain := os.Getenv("COOKIE_DOMAIN")

	sameSite := http.SameSiteLaxMode
	if isProduction {
		sameSite = http.SameSiteNoneMode
	}
	
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(duration),
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	}

	if cookieDomain != "" {
		cookie.Domain = cookieDomain
	}

    http.SetCookie(w, cookie)
}

func clearRefreshTokenCookie(w http.ResponseWriter) {
	isProduction := os.Getenv("APP_ENV") == "production"
	cookieDomain := os.Getenv("COOKIE_DOMAIN")

	sameSite := http.SameSiteLaxMode
	if isProduction {
		sameSite = http.SameSiteNoneMode
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	}

	if cookieDomain != "" {
		cookie.Domain = cookieDomain
	}

	http.SetCookie(w, cookie)
}

func extractUserIDFromClaims(claims map[string]any) (uint, error) {
	val, ok := claims["user_id"]
	if !ok {
		return 0, errors.New("invalid or missing user_id claim")
	}

	switch v := val.(type) {
	case float64:
		return uint(v), nil
	case uint:
		return v, nil
	case int:
		return uint(v), nil
	default:
		return 0, errors.New("invalid or missing user_id claim")
	}
}