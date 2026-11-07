package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Open(databaseURL string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres database: %w", err)
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var pingErr error
	for attempt := 0; attempt < 10; attempt++ {
		if pingErr = conn.PingContext(ctx); pingErr == nil {
			return conn, nil
		}
		select {
		case <-ctx.Done():
			conn.Close()
			return nil, fmt.Errorf("ping postgres database: %w", pingErr)
		case <-time.After(300 * time.Millisecond):
		}
	}
	conn.Close()
	return nil, fmt.Errorf("ping postgres database: %w", pingErr)
}
