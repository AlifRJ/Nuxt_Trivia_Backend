package middleware

import (
	"encoding/json"
	"net/http"
)

// JSONResponse defines a standardized JSON payload structure
type JSONResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// WriteJSONResponse writes a successful JSON response with data
func WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := JSONResponse{
		Success: statusCode >= 200 && statusCode < 300,
		Data:    data,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// WriteJSONError writes a standard JSON error response
func WriteJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := JSONResponse{
		Success: false,
		Error:   message,
	}

	_ = json.NewEncoder(w).Encode(resp)
}
