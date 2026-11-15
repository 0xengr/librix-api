package api

import (
	"github.com/jackc/pgx/v5"

	db "github.com/0xengr/librix-api/internal/database"
)

type QuerierWithTx interface {
	db.Querier
	WithTx(tx pgx.Tx) *db.Queries
}
