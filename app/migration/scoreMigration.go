package migration

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ScoreMigration(ctx context.Context, db *pgxpool.Pool) error {
	query := `
	-- Scores Table
	CREATE TABLE IF NOT EXISTS scores (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
		user_id UUID NOT NULL REFERENCES users(id) on DELETE CASCADE,
		score INT NOT NULL DEFAULT 0,
		category VARCHAR(100) NOT NULL,
		difficulty VARCHAR(20) NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
		deleted_at TIMESTAMPTZ
	);

	-- Fast User Query Index
	CREATE INDEX IF NOT EXISTS idx_scores_user_id 
	ON scores(user_id, created_at DESC) 
	WHERE deleted_at IS NULL;

	-- Global Leaderboard Index (Fast ORDER BY score DESC)
	CREATE INDEX IF NOT EXISTS idx_scores_global_leaderboard 
	ON scores(score DESC, id ASC) 
	WHERE deleted_at IS NULL;

	-- Category + Difficulty Leaderboard Index
	CREATE INDEX IF NOT EXISTS idx_scores_category_difficulty_leaderboard 
	ON scores(category, difficulty, score DESC) 
	WHERE deleted_at IS NULL;
	`

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to run scores migrations: %w", err)
	}
	return nil
}