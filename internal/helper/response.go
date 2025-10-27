package helper

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse defines the standard error structure for all APIs
type ErrorResponse struct {
	Errors map[string]string `json:"errors"`
}

// WriteErrorResponse writes a JSON response with proper status and format
func WriteErrorResponse(w http.ResponseWriter, statusCode int, fieldErrors map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := ErrorResponse{Errors: fieldErrors}
	json.NewEncoder(w).Encode(resp)
}

// WriteSimpleError is for non-field (generic) errors
func WriteSimpleError(w http.ResponseWriter, statusCode int, message string) {
	WriteErrorResponse(w, statusCode, map[string]string{
		"message": message,
	})
}
