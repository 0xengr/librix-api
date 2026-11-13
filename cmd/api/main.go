package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/0xengr/librix-api/internal/api"
	db "github.com/0xengr/librix-api/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {

	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("Missing Database URL")
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error(
			"unable to create connection pool",
			"error", err,
		)
		os.Exit(1)
	}

	// this ensures that the pool is closed when the application exits.
	defer pool.Close()

	// test if we can reach the database
	// successfully creating the pool doesn't
	// prove that DB is reachable, so we call
	// Ping() to verify connection.
	if err := pool.Ping(context.Background()); err != nil {
		slog.Error(
			"unable to connect a database",
			"error", err,
		)
		os.Exit(1)
	}

	queries := db.New(pool)
	bookHandler := api.NewBookHandler(queries)

	// chi router
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello world"))
	})

	r.Mount("/api/v1", bookHandler.Routes())

	slog.Info("server starting", "port", 3000)
	log.Fatal(http.ListenAndServe(":3000", r))
}
