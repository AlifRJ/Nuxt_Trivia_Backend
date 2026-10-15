package handler

import (
	"errors"
	"net/http"
	"uuid"

	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth/v5"
)

// ExtractUserIDFromClaims parses user_id safely whether unmarshaled as float64, uint, or int
func ExtractUserIDFromClaims(claims map[string]any) (uuid.UUID, error) {
	val, ok := claims["user_id"]
	if !ok {
		return uuid.Nil(), errors.New("invalid or missing user_id claim")
	}

	switch v := val.(type) {
	case uuid.UUID:
		return v, nil
	case string:
		id, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil(), errors.New("invalid string user_id claim")
		}
		return id, nil
	default:
		return uuid.Nil(), errors.New("unsupported user_id claim type")
	}
}

// GetUserIDRateKey provides a canonical rate-limiting key for httprate middleware
func GetUserIDRateKey(r *http.Request) (string, error) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		return httprate.CanonicalizeIP(r.RemoteAddr), nil
	}

	userID, err := ExtractUserIDFromClaims(claims)
	if err != nil {
		return httprate.CanonicalizeIP(r.RemoteAddr), nil
	}

	return userID.String(), nil
}