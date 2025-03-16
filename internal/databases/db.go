package databases

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
)

type DBService interface {
	// QueryRow executes a query that is expected to return at most one row.
	Transaction(ctx context.Context, txFunc func(*sql.Tx) (error, int)) (error, int)

	// GetDB returns the database connection
	GetDB() *sql.DB
}

type PostgresService struct {
	db *sql.DB
}

func NewPostgresService() *PostgresService {
	var err error
	var DB *sql.DB

	host := os.Getenv("HOST_POSTGRES")
	port := os.Getenv("PORT_POSTGRES")
	user := os.Getenv("USER_POSTGRES")
	password := os.Getenv("PASSWORD_POSTGRES")
	dbname := os.Getenv("DATABASE_POSTGRES")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s", host, port, user, password, dbname)

	if host == "localhost" {
		connStr += " sslmode=disable"
	}

	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	if err = DB.Ping(); err != nil {
		panic(err)
	}

	log.Println("Database connected")

	return &PostgresService{
		db: DB,
	}
}

func (p *PostgresService) GetDB() *sql.DB {
	return p.db
}

func (p *PostgresService) Transaction(ctx context.Context, txFunc func(*sql.Tx) (error, int)) (error, int) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err), fiber.StatusInternalServerError
	}

	// If fn returns an error, the transaction will be rolled back
	if err, statusCode := txFunc(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("error rolling back transaction: %v (original error: %w)", rbErr, err), fiber.StatusInternalServerError
		}

		return err, statusCode
	}

	// Commit transaction if fn was successful
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err), fiber.StatusInternalServerError
	}

	return nil, fiber.StatusOK
}
