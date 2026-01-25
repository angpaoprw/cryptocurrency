package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// Connect establishes a connection to the database
func Connect() *sql.DB {
	// Build connection string from environment variables
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "admin123"
	}

	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "tpl_crm"
	}

	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		// Default to 'require' for production, 'disable' for development
		env := os.Getenv("ENV")
		if env == "production" || env == "staging" {
			sslmode = "require"
		} else {
			sslmode = "disable"
		}
	}

	// Build connection string with SSL mode
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	log.Printf("Connecting to database with SSL mode: %s", sslmode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Printf("Successfully connected to database: %s@%s:%s/%s", user, host, port, dbname)

	return db
}

// Store provides all functions to execute db queries and transactions
type Store struct {
	*Queries
	db *sql.DB
}

// NewStore creates a new Store
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// GetDB returns the underlying database connection
func (store *Store) GetDB() *sql.DB {
	return store.db
}

// Ping checks if the database connection is alive
func (store *Store) Ping(ctx context.Context) error {
	return store.db.PingContext(ctx)
}

// ExecTx executes a function within a database transaction
func (store *Store) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
