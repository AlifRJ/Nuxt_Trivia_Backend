package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth/v5"
)

// ExtractUserIDFromClaims parses user_id safely whether unmarshaled as float64, uint, or int
func ExtractUserIDFromClaims(claims map[string]any) (uint, error) {
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
	case string:
		id, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, errors.New("invalid string user_id claim")
		}
		return uint(id), nil
	default:
		return 0, errors.New("unsupported user_id claim type")
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

	return strconv.FormatUint(uint64(userID), 10), nil
}