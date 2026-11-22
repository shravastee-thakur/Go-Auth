package utils

import (
	"encoding/json"
	"net/http"
)

// WriteJSON is your res.status(status).json(data) equivalent
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError sends a consistent JSON error message
func WriteError(w http.ResponseWriter, status int, message string)  {
	WriteJSON(w, status, map[string]string{"error": message})
}