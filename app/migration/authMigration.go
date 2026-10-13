package migration

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func AuthMigration(ctx context.Context, db *pgxpool.Pool) error {
	query := `
	-- 1. Auth Sessions Tabel (Refresh Token Persistence)
	CREATE TABLE IF NOT EXISTS auth_sessions (
		id SERIAL PRIMARY KEY,
		user_id INT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		refresh_token TEXT NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
	);

	-- 2. Index for searching by user_id
	CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id);
	`

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to run auth migrations: %w", err)
	}
	return nil
}