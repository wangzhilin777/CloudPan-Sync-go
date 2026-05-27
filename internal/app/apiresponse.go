package app

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeOK(w http.ResponseWriter, status int, data interface{}) {
	writeJSON(w, status, Envelope{
		OK:   true,
		Data: data,
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, Envelope{
		OK: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}
