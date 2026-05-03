package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"privacy-social-backend/internal/repository/db"
)

// Store provides all functions to execute db queries and transactions
type Store interface {
	db.Querier
	// Add transaction methods here later if needed
	ExecTx(ctx context.Context, fn func(*db.Queries) error) error
	GetDB() *sql.DB
}

// SQLStore provides all functions to execute SQL queries and transactions
type SQLStore struct {
	*db.Queries
	db *sql.DB
}

// NewStore creates a new Store
func NewStore(dbConn *sql.DB) Store {
	return &SQLStore{
		db:      dbConn,
		Queries: db.New(dbConn),
	}
}

// ExecTx executes a function within a database transaction
func (store *SQLStore) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	start := time.Now()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := db.New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	err = tx.Commit()
	
	duration := time.Since(start)
	if duration > 200*time.Millisecond {
		requestID, _ := ctx.Value("request_id").(string)
		log.Warn().
			Str("request_id", requestID).
			Dur("duration_ms", duration).
			Msg("Slow database transaction detected")
	}

	return err
}
func (store *SQLStore) GetDB() *sql.DB {
	return store.db
}
