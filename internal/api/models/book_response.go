package models

import (
	"time"

	db "github.com/0xengr/librix-api/internal/database"
)

// Here omitempty tag in description and year_of_publication
// means:
// - that a field is omitted from the JSON response when it
// - has it's zero value.

type BookResponse struct {
	ID                int32     `json:"id"`
	Title             string    `json:"title"`
	Author            string    `json:"author"`
	Description       string    `json:"description,omitempty"`
	YearOfPublication int32     `json:"year_of_publication,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func ToBookResponse(book db.Book) BookResponse {
	response := BookResponse{
		ID:        book.ID,
		Title:     book.Title,
		Author:    book.Author,
		CreatedAt: book.CreatedAt.Time,
		UpdatedAt: book.UpdatedAt.Time,
	}

	if book.Description.Valid {
		response.Description = book.Description.String
	}

	if book.YearOfPublication.Valid {
		response.YearOfPublication = book.YearOfPublication.Int32
	}

	return response

}
