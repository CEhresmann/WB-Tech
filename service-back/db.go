package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Postgres struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

var (
	PgInstance *Postgres
	pgOnce     sync.Once
)

func NewPG(ctx context.Context, connString string, logger *zap.Logger) (*Postgres, error) {
	pgOnce.Do(func() {
		db, err := pgxpool.New(ctx, connString)
		if err != nil {
			logger.Fatal("unable to create connection pool", zap.Error(err))
		}
		PgInstance = &Postgres{db: db, logger: logger}
	})

	return PgInstance, nil
}

func (pg *Postgres) Ping(ctx context.Context) error {
	return pg.db.Ping(ctx)
}

func (pg *Postgres) Close() {
	pg.db.Close()
	pg.logger.Info("database connection closed")
}

func (pg *Postgres) CreateTables(ctx context.Context) error {
	sqlScript, err := os.ReadFile("DB/initial.sql")
	if err != nil {
		return fmt.Errorf("failed to read SQL script: %w", err)
	}

	tx, err := pg.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	statements := strings.Split(string(sqlScript), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		_, err = tx.Exec(ctx, stmt)
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return fmt.Errorf("failed to rollback transaction: %w", rollbackErr)
			}
			return fmt.Errorf("failed to execute SQL statement: %s, error: %w", stmt, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
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

func connectToDB(dbConfig string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Connected to the database successfully")
	return pool, nil
}

type Config struct {
	DBConfig string `json:"DB_CONFIG"`
}

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
