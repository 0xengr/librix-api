package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/0xengr/librix-api/internal/api/models"
	db "github.com/0xengr/librix-api/internal/database"
)

// Client-facing API messages specific to users.
const (
	msgInvalidUserID = "invalid user id"
	msgUserNotFound  = "user not found"
)

// validation methods
func validateEmail(email string) error {
	email = strings.TrimSpace(email)

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return errors.New("invalid email address")
	}
	return nil
}

func validateNewUser(name, email string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(email) == "" {
		return errors.New("email is required")
	}
	return validateEmail(email)
}

// user structs for request validation
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (r CreateUserRequest) Validate() error {
	return validateNewUser(r.Name, r.Email)
}

type UpdateUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

func (r UpdateUserRequest) Validate() error {
	if r.Name == nil && r.Email == nil {
		return errors.New("at least one field is required")
	}
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return errors.New("name cannot be empty")
	}
	if r.Email != nil {
		if strings.TrimSpace(*r.Email) == "" {
			return errors.New("email cannot be empty")
		}
		return validateEmail(*r.Email)
	}
	return nil
}

type UserHandler struct {
	queries db.Querier
}

// a constructor function
func NewUserHandler(queries db.Querier) *UserHandler {
	return &UserHandler{
		queries: queries,
	}
}

// User handlers

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode create user request body", "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidRequestBody,
		)
		return
	}

	if err := req.Validate(); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	user, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Name:  strings.TrimSpace(req.Name),
		Email: strings.TrimSpace(req.Email),
	})

	if err != nil {
		if isUniqueViolation(err) {
			slog.Warn("unable to create user: email already exists", "error", err)
			writeError(
				w,
				http.StatusConflict,
				msgEmailAlreadyExists,
			)
			return
		}

		slog.Error("unable to create user", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		models.ToUserResponse(user),
	)
}

func (h *UserHandler) FetchAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		slog.Error("unable to list the users", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	responses := make([]models.UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, *models.ToUserResponse(user))
	}

	writeJSON(
		w,
		http.StatusOK,
		responses,
	)
}

func (h *UserHandler) FetchUserByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		slog.Error(msgInvalidUserID, "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidUserID,
		)
		return
	}

	user, err := h.queries.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Error(msgUserNotFound, "error", err)
			writeError(
				w,
				http.StatusNotFound,
				msgUserNotFound,
			)
			return
		}
		slog.Error("unable to get the user", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		models.ToUserResponse(user),
	)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		slog.Error(msgInvalidUserID, "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidUserID,
		)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode update user request body", "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidRequestBody,
		)
		return
	}

	if err := req.Validate(); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	params := db.UpdateUserParams{
		ID: id,
	}
	if req.Name != nil {
		params.Name = pgtype.Text{
			String: strings.TrimSpace(*req.Name),
			Valid:  true,
		}
	}
	if req.Email != nil {
		params.Email = pgtype.Text{
			String: strings.TrimSpace(*req.Email),
			Valid:  true,
		}
	}

	user, err := h.queries.UpdateUser(r.Context(), params)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Error(msgUserNotFound, "error", err)
			writeError(
				w,
				http.StatusNotFound,
				msgUserNotFound,
			)
			return
		}

		if isUniqueViolation(err) {
			slog.Warn("unable to update user: email already exists", "error", err)
			writeError(
				w,
				http.StatusConflict,
				msgEmailAlreadyExists,
			)
			return
		}

		slog.Error("unable to update user", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	writeJSON(w, http.StatusOK, models.ToUserResponse(user))
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		slog.Error(msgInvalidUserID, "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidUserID,
		)
		return
	}

	_, err = h.queries.DeleteUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Error(msgUserNotFound, "error", err)
			writeError(
				w,
				http.StatusNotFound,
				msgUserNotFound,
			)
			return
		}

		slog.Error("unable to delete user", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

// Routes() creates a chi router and registers
// all user-related endpoints on it
func (h *UserHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.FetchAllUsers)
	r.Get("/{id}", h.FetchUserByID)
	r.Post("/", h.CreateUser)
	r.Put("/{id}", h.UpdateUser)
	r.Delete("/{id}", h.DeleteUser)

	return r
}
