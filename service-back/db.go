package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type postgres struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

var (
	pgInstance *postgres
	pgOnce     sync.Once
)

// NewPG initializes a new PostgreSQL connection pool and returns a postgres instance.
func NewPG(ctx context.Context, connString string, logger *zap.Logger) (*postgres, error) {
	pgOnce.Do(func() {
		db, err := pgxpool.New(ctx, connString)
		if err != nil {
			logger.Fatal("unable to create connection pool", zap.Error(err))
		}
		pgInstance = &postgres{db: db, logger: logger}
	})

	return pgInstance, nil
}

// Ping checks the connection to the database.
func (pg *postgres) Ping(ctx context.Context) error {
	return pg.db.Ping(ctx)
}

// Close closes the database connection pool.
func (pg *postgres) Close() {
	pg.db.Close()
	pg.logger.Info("database connection closed")
}

// CreateTables creates tables in the database based on the provided SQL script.
func (pg *postgres) CreateTables(ctx context.Context) error {
	sqlScript, err := os.ReadFile("../DB/initial.sql")
	if err != nil {
		return fmt.Errorf("failed to read SQL script: %w", err)
	}

	_, err = pg.db.Exec(ctx, string(sqlScript))
	if err != nil {
		return fmt.Errorf("failed to execute SQL script: %w", err)
	}

	pg.logger.Info("Tables created or already exist")
	return nil
}

func LoadCacheFromDB(dbConfig string, ctx context.Context) error {
	db, err := connectToDB(dbConfig)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(ctx, "SELECT order_uid FROM orders") // Adjust the query as needed
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.OrderUID); err != nil {
			return err
		}
		cacheMu.Lock()
		cache[order.OrderUID] = order
		cacheMu.Unlock()
	}

	return nil
}

// connectToDB establishes a connection to the PostgreSQL database using the provided connection string.
func connectToDB(dbConfig string) (*pgxpool.Pool, error) {
	// Create a connection pool
	pool, err := pgxpool.New(context.Background(), dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify the connection by executing a simple query
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Connected to the database successfully")
	return pool, nil
}

// Config holds the database configuration
type Config struct {
	DBConfig string `json:"DB_CONFIG"`
}

// LoadConfig reads the configuration from a JSON file
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}
