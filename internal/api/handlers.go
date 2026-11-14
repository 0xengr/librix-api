package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/0xengr/librix-api/internal/api/models"
)

func writeJSON(w http.ResponseWriter, status int, data any) {

	if data != nil {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(status)

	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}

}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{
		Error: message,
	})
}

func parseID(r *http.Request) (int32, error) {

	idStr := chi.URLParam(r, "id")

	value, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		return 0, err
	}

	if value <= 0 {
		return 0, errors.New("id must be a positive number")
	}

	return int32(value), nil
}
