package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Error is an API error that is written as {"error": {"code", "message"}}.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// NewError creates an API error.
func NewError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// HandlerFunc is an HTTP handler that may return an error.
// A returned *Error is written as is; any other error becomes a 500.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err == nil {
		return
	}
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		apiErr = NewError(http.StatusInternalServerError, "internal", "Internal server error")
	}
	WriteError(w, apiErr)
}

// WriteJSON writes v as a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writing JSON response", "error", err)
	}
}

// WriteError writes an API error response.
func WriteError(w http.ResponseWriter, e *Error) {
	type body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	WriteJSON(w, e.Status, map[string]body{"error": {Code: e.Code, Message: e.Message}})
}
