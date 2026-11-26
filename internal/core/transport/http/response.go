package http

import (
	"encoding/json"
	"errors"
	"net/http"

	coreerrors "github.com/makmanu/golang_todoapp/internal/core/errors"
	"github.com/makmanu/golang_todoapp/internal/core/logger"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append(data, '\n'))
}
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	switch {
	case errors.Is(err, coreerrors.ErrInvalidArgument):
		status = http.StatusBadRequest
		message = "invalid argument"
	case errors.Is(err, coreerrors.ErrNotFound):
		status = http.StatusNotFound
		message = "not found"
	case errors.Is(err, coreerrors.ErrConflict):
		status = http.StatusConflict
		message = "conflict"
	}
	if status == http.StatusInternalServerError {
		logger.FromContext(r.Context()).Error("request failed", "error", err)
	}
	WriteJSON(w, status, ErrorResponse{Error: message})
}
