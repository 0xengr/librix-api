package models

import (
	"time"

	db "github.com/0xengr/librix-api/internal/database"
)

type BorrowRecordResponse struct {
	ID         int32     `json:"id"`
	BookID     int32     `json:"book_id"`
	UserID     int32     `json:"user_id"`
	BorrowedAt time.Time `json:"borrowed_at"`
	DueDate    time.Time `json:"due_date"`
	ReturnedAt time.Time `json:"returned_at"`
}

func ToBorrowRecordResponse(borrowedBook db.BorrowedBook) BorrowRecordResponse {

	return BorrowRecordResponse{
		ID:         borrowedBook.ID,
		BookID:     borrowedBook.BookID,
		UserID:     borrowedBook.UserID,
		BorrowedAt: borrowedBook.BorrowedAt.Time,
		DueDate:    borrowedBook.DueDate.Time,
		ReturnedAt: borrowedBook.ReturnedAt.Time,
	}

}
