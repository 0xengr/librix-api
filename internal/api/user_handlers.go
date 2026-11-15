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
	// Step 1: Decode the JSON body into a create request.
	var req CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Malformed JSON or a field with the wrong type -> 400.
		slog.Error("failed to decode create user request body", "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidRequestBody,
		)
		return
	}

	// Step 2: Validate the required fields (name and email).
	if err := req.Validate(); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	// Step 3: Insert the user, trimming whitespace around name and email.
	user, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Name:  strings.TrimSpace(req.Name),
		Email: strings.TrimSpace(req.Email),
	})

	if err != nil {
		// Step 4: A unique-constraint violation means the email is taken -> 409.
		if isUniqueViolation(err) {
			slog.Warn("unable to create user: email already exists", "error", err)
			writeError(
				w,
				http.StatusConflict,
				msgEmailAlreadyExists,
			)
			return
		}

		// Step 5: Any other database failure -> 500.
		slog.Error("unable to create user", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	// Step 6: Respond with 201 Created and the new user.
	writeJSON(
		w,
		http.StatusCreated,
		models.ToUserResponse(user),
	)
}

func (h *UserHandler) FetchAllUsers(w http.ResponseWriter, r *http.Request) {
	// Step 1: Load every user, ordered by id.
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

	// Step 2: Map the database rows to API responses.
	responses := make([]models.UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, *models.ToUserResponse(user))
	}

	// Step 3: Respond with 200. An empty table returns [] instead of null.
	writeJSON(
		w,
		http.StatusOK,
		responses,
	)
}

func (h *UserHandler) FetchUserByID(w http.ResponseWriter, r *http.Request) {
	// Step 1: Parse and validate the {id} path parameter.
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

	// Step 2: Fetch the user by id.
	user, err := h.queries.GetUser(r.Context(), id)
	if err != nil {
		// Step 3: No row for this id -> 404; any other failure -> 500.
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

	// Step 4: Respond with the user.
	writeJSON(
		w,
		http.StatusOK,
		models.ToUserResponse(user),
	)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Step 1: Parse and validate the {id} path parameter.
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

	// Step 2: Decode the JSON body. Name and Email are pointers so we can tell
	// which fields the client actually sent (missing fields stay nil).
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

	// Step 3: Validate the partial update: at least one field, non-empty
	// values, and a valid email.
	if err := req.Validate(); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	// Step 4: Build the update params, setting only the fields that were sent.
	// The SQL COALESCE keeps the existing value for nil fields, so this is a
	// partial update rather than a full replace.
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

	// Step 5: Apply the update and get the updated user back.
	user, err := h.queries.UpdateUser(r.Context(), params)

	if err != nil {
		// Step 6: No row for this id -> 404.
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Error(msgUserNotFound, "error", err)
			writeError(
				w,
				http.StatusNotFound,
				msgUserNotFound,
			)
			return
		}

		// Step 7: The new email is already used -> 409.
		if isUniqueViolation(err) {
			slog.Warn("unable to update user: email already exists", "error", err)
			writeError(
				w,
				http.StatusConflict,
				msgEmailAlreadyExists,
			)
			return
		}

		// Step 8: Any other database failure -> 500.
		slog.Error("unable to update user", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	// Step 9: Respond with the updated user.
	writeJSON(w, http.StatusOK, models.ToUserResponse(user))
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Step 1: Parse and validate the {id} path parameter.
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

	// Step 2: Delete the row. The returned id is not needed by the response.
	_, err = h.queries.DeleteUser(r.Context(), id)
	if err != nil {
		// Step 3: No row for this id -> 404; any other failure -> 500.
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

	// Step 4: Respond with 204 No Content.
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *UserHandler) BorrowHistory(w http.ResponseWriter, r *http.Request) {
	// Step 1: Parse and validate the {id} path parameter (the user id).
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

	// Step 2: Make sure the user exists so an unknown id returns 404 instead of
	// an empty list.
	_, err = h.queries.GetUser(r.Context(), id)
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

		slog.Error("failed to get the user", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	// Step 3: Load every loan for this user, both active and returned,
	// newest first.
	records, err := h.queries.GetUserBorrowHistory(r.Context(), id)
	if err != nil {
		slog.Error("failed to get user borrow history", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	// Step 4: Map the joined rows (loan + book title/author) to responses.
	responses := make([]models.HistoryRowResponse, 0, len(records))

	for _, record := range records {
		responses = append(responses, models.ToHistoryRowResponse(record))
	}

	// Step 5: Respond with 200. A user with no loans returns [] instead of null.
	writeJSON(w, http.StatusOK, responses)
}

// Routes() creates a chi router and registers
// all user-related endpoints on it
func (h *UserHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// User routes
	r.Get("/", h.FetchAllUsers)
	r.Get("/{id}", h.FetchUserByID)
	r.Post("/", h.CreateUser)
	r.Put("/{id}", h.UpdateUser)
	r.Delete("/{id}", h.DeleteUser)

	// Borrow history route
	r.Get("/{id}/history", h.BorrowHistory)

	return r
}
