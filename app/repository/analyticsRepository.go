package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository interface {
	GetExecutiveKPIs(ctx context.Context) (totalGames int64, totalPlayers int64, avgScore float64, err error)
	GetCategoryStats(ctx context.Context, limit int) ([]CategoryStat, error)
	GetDifficultyStats(ctx context.Context) ([]DifficultyStat, error)
	GetDailyTrends(ctx context.Context, days int) ([]DailyTrendStat, error)
}

type analyticsRepo struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepository(db *pgxpool.Pool) AnalyticsRepository {
	return &analyticsRepo{db: db}
}

// Data Transfer Objects (DTOs)
type CategoryStat struct {
	Category    string  `json:"category"`
	GamesPlayed int64   `json:"games_played"`
	AvgScore    float64 `json:"avg_score"`
}

type DifficultyStat struct {
	Difficulty string  `json:"difficulty"`
	TotalGames int64   `json:"total_games"`
	AvgScore   float64 `json:"avg_score"`
}

type DailyTrendStat struct {
	Date        string  `json:"date"`
	GamesPlayed int64   `json:"games_played"`
	AvgScore    float64 `json:"avg_score"`
}

func (r *analyticsRepo) GetExecutiveKPIs(ctx context.Context) (int64, int64, float64, error) {
	query := `
		SELECT 
			COUNT(*)::bigint AS total_games, 
			COUNT(DISTINCT user_id)::bigint AS total_players, 
			COALESCE(ROUND(AVG(score), 1), 0.0)::float8 AS avg_score
		FROM scores 
		WHERE deleted_at IS NULL;`

	var totalGames, totalPlayers int64
	var avgScore float64

	err := r.db.QueryRow(ctx, query).Scan(&totalGames, &totalPlayers, &avgScore)
	if err != nil && err != pgx.ErrNoRows {
		return 0, 0, 0, fmt.Errorf("repository: failed to fetch KPIs: %w", err)
	}

	return totalGames, totalPlayers, avgScore, nil
}

func (r *analyticsRepo) GetCategoryStats(ctx context.Context, limit int) ([]CategoryStat, error) {
	query := `
		SELECT 
			category, 
			COUNT(*)::bigint AS games_played, 
			COALESCE(ROUND(AVG(score), 1), 0.0)::float8 AS avg_score
		FROM scores
		WHERE deleted_at IS NULL
		GROUP BY category
		ORDER BY games_played DESC
		LIMIT $1;`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to query category stats: %w", err)
	}
	defer rows.Close()

	stats := make([]CategoryStat, 0)
	for rows.Next() {
		var cs CategoryStat
		if err := rows.Scan(&cs.Category, &cs.GamesPlayed, &cs.AvgScore); err != nil {
			return nil, fmt.Errorf("repository: failed scanning category row: %w", err)
		}
		stats = append(stats, cs)
	}

	return stats, rows.Err()
}

func (r *analyticsRepo) GetDifficultyStats(ctx context.Context) ([]DifficultyStat, error) {
	query := `
		SELECT 
			difficulty, 
			COUNT(*)::bigint AS total_games,
			COALESCE(ROUND(AVG(score), 1), 0.0)::float8 AS avg_score
		FROM scores
		WHERE deleted_at IS NULL
		GROUP BY difficulty
		ORDER BY CASE LOWER(difficulty) 
			WHEN 'easy' THEN 1 
			WHEN 'medium' THEN 2 
			WHEN 'hard' THEN 3 
			ELSE 4 
		END;`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to query difficulty stats: %w", err)
	}
	defer rows.Close()

	stats := make([]DifficultyStat, 0)
	for rows.Next() {
		var ds DifficultyStat
		if err := rows.Scan(&ds.Difficulty, &ds.TotalGames, &ds.AvgScore); err != nil {
			return nil, fmt.Errorf("repository: failed scanning difficulty row: %w", err)
		}
		stats = append(stats, ds)
	}

	return stats, rows.Err()
}

func (r *analyticsRepo) GetDailyTrends(ctx context.Context, days int) ([]DailyTrendStat, error) {
	query := `
		SELECT 
			TO_CHAR(created_at, 'YYYY-MM-DD') AS play_date,
			COUNT(*)::bigint AS games_played,
			COALESCE(ROUND(AVG(score), 1), 0.0)::float8 AS avg_score
		FROM scores
		WHERE deleted_at IS NULL 
		  AND created_at >= CURRENT_DATE - ($1 || ' days')::INTERVAL
		GROUP BY play_date
		ORDER BY play_date ASC;`

	rows, err := r.db.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to query daily trends: %w", err)
	}
	defer rows.Close()

	stats := make([]DailyTrendStat, 0)
	for rows.Next() {
		var ts DailyTrendStat
		if err := rows.Scan(&ts.Date, &ts.GamesPlayed, &ts.AvgScore); err != nil {
			return nil, fmt.Errorf("repository: failed scanning trend row: %w", err)
		}
		stats = append(stats, ts)
	}

	return stats, rows.Err()
}
