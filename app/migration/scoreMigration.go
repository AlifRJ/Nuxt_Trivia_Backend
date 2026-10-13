package migration

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ScoreMigration(ctx context.Context, db *pgxpool.Pool) error {
	query := `
	-- 1. Scores Table
	CREATE TABLE IF NOT EXISTS scores (
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) on DELETE CASCADE,
		score INT NOT NULL DEFAULT 0,
		category VARCHAR(255) NOT NULL,
		difficulty VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
		deleted_at TIMESTAMP
	);

	-- 2. Fast User Query Index
	CREATE INDEX IF NOT EXISTS idx_scores_user_id 
	ON scores(user_id) 
	WHERE deleted_at IS NULL;

	-- 3. Global Leaderboard Index (Fast ORDER BY score DESC)
	CREATE INDEX IF NOT EXISTS idx_scores_global_leaderboard 
	ON scores(score DESC, id ASC) 
	WHERE deleted_at IS NULL;

	-- 4. Category + Difficulty Leaderboard Index
	CREATE INDEX IF NOT EXISTS idx_scores_category_difficulty_leaderboard 
	ON scores(category, difficulty, score DESC) 
	WHERE deleted_at IS NULL;
	`

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to run scores migrations: %w", err)
	}
	return nil
}