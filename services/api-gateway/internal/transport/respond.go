package transport

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code string `json:"code"`
}

func RespondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("Failed to encode response payload", "error", err)
	}
}

func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, ErrorResponse{
		Error: message,
		Code:  strconv.Itoa(status),
	})
}

