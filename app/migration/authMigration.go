package migration

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func AuthMigration(ctx context.Context, db *pgxpool.Pool) error {
	query := `
	-- Auth Sessions Tabel (Refresh Token Persistence)
	CREATE TABLE IF NOT EXISTS auth_sessions (
		id UUID PRIMARY KEY DEFAULT uuidv7(),
		user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
		refresh_token TEXT NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
	);

	-- Index for searching by user_id
	CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id);

	-- Quick lookup by refresh token
	-- CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_sessions_refresh_token ON auth_sessions(refresh_token);
	`

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to run auth migrations: %w", err)
	}
	return nil
}