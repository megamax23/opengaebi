package db

import (
	"context"
	"fmt"
)

func New(ctx context.Context, dbType, url string) (DB, error) {
	switch dbType {
	case "sqlite":
		return NewSQLite(url)
	case "postgres":
		return NewPostgres(ctx, url)
	default:
		return nil, fmt.Errorf("unsupported db type: %s (use 'sqlite' or 'postgres')", dbType)
	}
}
