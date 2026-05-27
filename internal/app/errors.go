package app

import "net/http"

type HTTPError struct {
	Status  int
	Code    string
	Message string
}

func (e *HTTPError) Error() string {
	return e.Code + ": " + e.Message
}

func handleError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	if httpErr, ok := err.(*HTTPError); ok {
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error.")
}
