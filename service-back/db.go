package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postgres struct {
	db *pgxpool.Pool
}

var (
	pgInstance *postgres
	pgOnce     sync.Once
)

func NewPG(ctx context.Context, connString string) (*postgres, error) {
	pgOnce.Do(func() {
		db, err := pgxpool.New(ctx, connString)
		if err != nil {
			log.Fatalf("unable to create connection pool: %v", err)
		}
		pgInstance = &postgres{db}
	})

	return pgInstance, nil
}

func (pg *postgres) Ping(ctx context.Context) error {
	return pg.db.Ping(ctx)
}

func (pg *postgres) Close() {
	pg.db.Close()
}

func (pg *postgres) CreateTables(ctx context.Context) error {
	sqlScript, err := ioutil.ReadFile("../DB/initial.sql")
	if err != nil {
		return fmt.Errorf("failed to read SQL script: %w", err)
	}

	_, err = pg.db.Exec(ctx, string(sqlScript))
	if err != nil {
		return fmt.Errorf("failed to execute SQL script: %w", err)
	}

	log.Println("Tables created or already exist")
	return nil
}
