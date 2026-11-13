package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/0xengr/librix-api/internal/api/models"
	db "github.com/0xengr/librix-api/internal/database"
)

// Client-facing API messages.
const (
	msgInternalServerError = "internal server error"
	msgInvalidRequestBody  = "invalid request body"
	msgInvalidBookID       = "invalid book id"
	msgBookNotFound        = "book not found"
)

// validation method
func validateBook(title, author string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(author) == "" {
		return errors.New("author is required")
	}
	return nil
}

// book structs for request validation
type CreateBookRequest struct {
	Title             string `json:"title"`
	Author            string `json:"author"`
	Description       string `json:"description"`
	YearOfPublication int32  `json:"year_of_publication"`
}

func (r CreateBookRequest) Validate() error {
	return validateBook(r.Title, r.Author)
}

type UpdateBookRequest struct {
	Title             string `json:"title"`
	Author            string `json:"author"`
	Description       string `json:"description"`
	YearOfPublication int32  `json:"year_of_publication"`
}

func (r UpdateBookRequest) Validate() error {
	return validateBook(r.Title, r.Author)
}

type BookHandler struct {
	queries db.Querier
}

// a constructor function
func NewBookHandler(queries db.Querier) *BookHandler {
	return &BookHandler{
		queries: queries,
	}
}

// Book handlers

func (h *BookHandler) CreateBook(w http.ResponseWriter, r *http.Request) {
	var req CreateBookRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode create book request body", "error", err)
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

	params := db.CreateBookParams{
		Title:  req.Title,
		Author: req.Author,
	}

	if req.Description != "" {
		params.Description = pgtype.Text{
			String: req.Description,
			Valid:  true,
		}
	}

	if req.YearOfPublication != 0 {
		params.YearOfPublication = pgtype.Int4{
			Int32: req.YearOfPublication,
			Valid: true,
		}
	}

	book, err := h.queries.CreateBook(r.Context(), params)
	if err != nil {
		slog.Error("unable to create book", "error", err)
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
		models.ToBookResponse(book),
	)
}

func (h *BookHandler) FetchBooks(w http.ResponseWriter, r *http.Request) {
	books, err := h.queries.ListBooks(r.Context())
	if err != nil {
		slog.Error("unable to list the books", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	responses := make([]models.BookResponse, 0, len(books))
	for _, book := range books {
		responses = append(responses, models.ToBookResponse(book))
	}

	writeJSON(
		w,
		http.StatusOK,
		responses,
	)
}

func (h *BookHandler) FetchBookByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)

	if err != nil {
		slog.Error(msgInvalidBookID, "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidBookID,
		)
		return
	}

	book, err := h.queries.GetBook(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Error(msgBookNotFound, "error", err)
			writeError(
				w,
				http.StatusNotFound,
				msgBookNotFound,
			)
			return
		}
		slog.Error("unable to get the book", "error", err)
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
		models.ToBookResponse(book),
	)
}

func (h *BookHandler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		slog.Error(msgInvalidBookID, "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidBookID,
		)
		return
	}

	var req UpdateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode update book request body", "error", err)
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

	params := db.UpdateBookParams{
		ID:     int32(id),
		Title:  req.Title,
		Author: req.Author,
	}

	if req.Description != "" {
		params.Description = pgtype.Text{
			String: req.Description,
			Valid:  true,
		}
	}

	if req.YearOfPublication != 0 {
		params.YearOfPublication = pgtype.Int4{
			Int32: req.YearOfPublication,
			Valid: true,
		}
	}

	book, err := h.queries.UpdateBook(r.Context(), params)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Error(msgBookNotFound, "error", err)
			writeError(
				w,
				http.StatusNotFound,
				msgBookNotFound,
			)
			return
		}

		slog.Error("unable to update book", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	writeJSON(w, http.StatusOK, models.ToBookResponse(book))
}

func (h *BookHandler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		slog.Error(msgInvalidBookID, "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidBookID,
		)
		return
	}

	_, err = h.queries.DeleteBook(r.Context(), id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Error(msgBookNotFound, "error", err)
			writeError(
				w,
				http.StatusNotFound,
				msgBookNotFound,
			)
			return
		}

		slog.Error("unable to delete book", "error", err)
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
// all book-related endpoints on it
func (h *BookHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/books", h.FetchBooks)
	r.Get("/books/{id}", h.FetchBookByID)
	r.Post("/books", h.CreateBook)
	r.Put("/books/{id}", h.UpdateBook)
	r.Delete("/books/{id}", h.DeleteBook)

	return r
}
