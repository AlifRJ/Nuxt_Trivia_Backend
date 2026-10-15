package migration

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func UserMigration(ctx context.Context, db *pgxpool.Pool) error {
	query := `
	-- Users Tabel
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT uuidv7(),
		name VARCHAR(255) NOT NULL,
		username VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL,
		password VARCHAR(255) NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
		updated_at TIMESTAMPTZ,
		deleted_at TIMESTAMPTZ
	);

	-- Email/Username Index for Login & Soft Delete
	CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;
	CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE deleted_at IS NULL;
	`

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to run user migrations: %w", err)
	}
	return nil
}