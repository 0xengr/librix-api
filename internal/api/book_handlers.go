package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/0xengr/librix-api/internal/api/models"
	db "github.com/0xengr/librix-api/internal/database"
)

// Client-facing API messages specific to books.
const (
	msgInvalidBookID = "invalid book id"
	msgBookNotFound  = "book not found"
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

type BorrowBookRequest struct {
	BookID int32 `json:"book_id"`
	UserID int32 `json:"user_id"`
	Days   int32 `json:"days"`
}

func (r BorrowBookRequest) Validate() error {
	if r.BookID <= 0 {
		return errors.New("book_id must be a positive number")
	}
	if r.UserID <= 0 {
		return errors.New("user_id must be a positive number")
	}
	if r.Days < 0 {
		return errors.New("days cannot be negative")
	}
	return nil
}

type ReturnBookRequest struct {
	BookID int32 `json:"book_id"`
	UserID int32 `json:"user_id"`
}

func (r ReturnBookRequest) Validate() error {
	if r.BookID <= 0 {
		return errors.New("book_id must be a positive number")
	}
	if r.UserID <= 0 {
		return errors.New("user_id must be a positive number")
	}
	return nil
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
		ID:     id,
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

func (h *BookHandler) BorrowBook(w http.ResponseWriter, r *http.Request) {
	// Step 1: Decode the JSON body (book_id, user_id, days) into a request struct.
	var req BorrowBookRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Malformed JSON or a field with the wrong type is the client's fault -> 400.
		slog.Error("failed to decode the r.Body", "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidRequestBody,
		)
		return
	}

	// Step 2: Validate the decoded values.
	// book_id and user_id must be positive; days must not be negative.
	if err := req.Validate(); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	// Step 3: Apply the default loan period.
	// A value of 0 Days is allowed; it is treated as the 14 day loan period.
	if req.Days == 0 {
		req.Days = 14
	}

	// Step 4: Check if the book exists and load its current state.
	book, err := h.queries.GetBook(r.Context(), req.BookID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No row for this id -> the book does not exist -> 404.
			slog.Error("book not found", "error", err)
			writeError(
				w,
				http.StatusNotFound,
				msgBookNotFound,
			)
			return
		}

		// Any other database failure is our problem -> 500.
		slog.Error("failed to fetch the book", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	// Step 5: Reject the loan when the book is already borrowed.
	if !book.Available {
		// The book exists but cannot be borrowed right now -> 409 Conflict.
		writeError(
			w,
			http.StatusConflict,
			"book is not available for borrowing",
		)
		return
	}

	// Step 6: Check that the user exists (the discarded value is only used as
	// an existence check; the foreign key would otherwise fail later).
	_, err = h.queries.GetUser(r.Context(), req.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No row for this id -> the user does not exist -> 404.
			writeError(
				w,
				http.StatusNotFound,
				msgUserNotFound,
			)
			return
		}

		// Any other database failure -> 500.
		slog.Error("failed to get the User", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	// Step 7: Calculate the due date as "now + requested days".
	dueDate := time.Now().Add(time.Duration(req.Days) * 24 * time.Hour)

	// Step 8: Insert the loan row into borrowed_books and get it back
	// (including the database-generated id and borrowed_at).
	borrowRecord, err := h.queries.BorrowBook(r.Context(), db.BorrowBookParams{
		BookID:  req.BookID,
		UserID:  req.UserID,
		DueDate: pgtype.Timestamptz{Time: dueDate, Valid: true},
	})

	if err != nil {
		slog.Error("failed to borrow book", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	// Step 9: Mark the book as unavailable so it cannot be borrowed again.
	err = h.queries.UpdateBookAvailability(r.Context(), db.UpdateBookAvailabilityParams{
		ID:        req.BookID,
		Available: false,
	})

	if err != nil {
		slog.Error("failed to update the book availability", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	// Step 10: Return the created loan record to the client.
	writeJSON(w, http.StatusOK, models.ToBorrowRecordResponse(borrowRecord))

}

func (h *BookHandler) ReturnBook(w http.ResponseWriter, r *http.Request) {

	var req ReturnBookRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode r.Body", "error", err)
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

	// Return the book
	returnRecord, err := h.queries.ReturnBook(r.Context(), db.ReturnBookParams{
		BookID: req.BookID,
		UserID: req.UserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(
				w,
				http.StatusNotFound,
				"no active borrow record found",
			)
			return
		}

		slog.Error("failed to retrieve borrow record", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	// Mark book as available
	err = h.queries.UpdateBookAvailability(r.Context(), db.UpdateBookAvailabilityParams{
		ID:        req.BookID,
		Available: true,
	})

	if err != nil {
		slog.Error("failed to update the book availability", "error", err)
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
		models.ToReturnRecordResponse(returnRecord, "book returned successfully"),
	)

}

func (h *BookHandler) GetUserBorrowedBooks(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		slog.Error("invalid user ID", "error", err)
		writeError(
			w,
			http.StatusBadRequest,
			msgInvalidUserID,
		)
		return
	}

	books, err := h.queries.GetUserBorrowedBooks(r.Context(), id)
	if err != nil {
		slog.Error("failed to get user borrowed books", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	var response = make([]models.BorrowedBooksRowResponse, 0, len(books))

	for _, book := range books {
		response = append(response, models.ToBorrowedBooksRowResponse(book))
	}

	writeJSON(w, http.StatusOK, response)

}

func (h *BookHandler) GetOverdueBooks(w http.ResponseWriter, r *http.Request) {
	books, err := h.queries.GetOverDueBooks(r.Context())
	if err != nil {
		slog.Error("failed to get overdue books", "error", err)
		writeError(
			w,
			http.StatusInternalServerError,
			msgInternalServerError,
		)
		return
	}

	var response = make([]models.OverdueBooksRowResponse, 0, len(books))

	for _, book := range books {
		response = append(response, models.ToOverdueBooksRowResponse(book))
	}

	writeJSON(w, http.StatusOK, response)

}

// Routes() creates a chi router and registers
// all book-related endpoints on it
func (h *BookHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Book routes
	r.Get("/", h.FetchBooks)
	r.Get("/{id}", h.FetchBookByID)
	r.Post("/", h.CreateBook)
	r.Put("/{id}", h.UpdateBook)
	r.Delete("/{id}", h.DeleteBook)

	// Borrow and return Routes
	r.Post("/borrow", h.BorrowBook)
	r.Post("/return", h.ReturnBook)

	// User borrowed books route
	r.Get("/user/{id}/borrowed", h.GetUserBorrowedBooks)

	// Overdue books route
	r.Get("/overdue", h.GetOverdueBooks)

	return r
}
