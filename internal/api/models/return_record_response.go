package models

import db "github.com/0xengr/librix-api/internal/database"

type ReturnRecordResponse struct {
	ID      int32  `json:"id"`
	BookID  int32  `json:"book_id"`
	UserID  int32  `json:"user_id"`
	Message string `json:"message"`
}

func ToReturnRecordResponse(record db.BorrowedBook, message string) ReturnRecordResponse {
	return ReturnRecordResponse{
		ID:      record.ID,
		BookID:  record.BookID,
		UserID:  record.UserID,
		Message: message,
	}
}
