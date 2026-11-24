package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/AlifRJ/Go_Auth/app/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrScoreNotFound = errors.New("score not found")
)

type PostgresScoreRepository struct {
	db *pgxpool.Pool
}

func NewPostgresScoreRepository(db *pgxpool.Pool) model.ScoreRepository {
	return &PostgresScoreRepository{db: db}
}

func (r *PostgresScoreRepository) GetAll(ctx context.Context, filter model.ScoreFilter) ([]*model.ScoreWithUsername, int, error) {
	var conditions []string
	args := pgx.NamedArgs{
		"limit":  filter.Limit,
		"offset": filter.Offset,
	}

	// Trashed filter
	switch filter.Trashed {
	case "only":
		conditions = append(conditions, "scores.deleted_at IS NOT NULL")
	case "with":
		// Include both active and soft-deleted scores
	default:
		conditions = append(conditions, "scores.deleted_at IS NULL")
	}

	// Search filter
	if filter.Search != "" {
		conditions = append(conditions, `(
			scores.id::text ILIKE @search OR 
			users.username ILIKE @search OR
			scores.category ILIKE @search OR 
			scores.difficulty ILIKE @search
		)`)
		args["search"] = "%" + filter.Search + "%"
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT 
			scores.id, scores.user_id, users.username, scores.score, scores.category, scores.difficulty, scores.created_at, scores.deleted_at,
			COUNT(*) OVER() AS total_count
		FROM scores
		LEFT JOIN users ON scores.user_id = users.id
		%s
		ORDER BY scores.created_at DESC, scores.id DESC
		LIMIT @limit OFFSET @offset
	`, whereClause)

	rows, err := r.db.Query(ctx, query, args)
	if err != nil {
		return nil, 0, fmt.Errorf("query GetAll failed: %w", err)
	}
	defer rows.Close()

	total := 0
	scores, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*model.ScoreWithUsername, error) {
		var s model.ScoreWithUsername
		err := row.Scan(
			&s.ID,
			&s.UserID,
			&s.Username,
			&s.Score,
			&s.Category,
			&s.Difficulty,
			&s.CreatedAt,
			&s.DeletedAt,
			&total,
		)
		return &s, err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("collect GetAll rows failed: %w", err)
	}

	// Fallback count if offset is past the last page (so total stays accurate)
	if len(scores) == 0 && filter.Offset > 0 {
		countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM scores %s`, whereClause)
		if err := r.db.QueryRow(ctx, countQuery, args).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("fallback count GetAll failed: %w", err)
		}
	}

	return scores, total, nil
}

func (r *PostgresScoreRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Score, error) {
	query := `SELECT id, user_id, score, category, difficulty, created_at, deleted_at FROM scores WHERE deleted_at IS NULL AND id = $1`

	var score model.Score
	err := r.db.QueryRow(ctx, query, id).Scan(&score.ID, &score.UserID, &score.Score, &score.Category, &score.Difficulty, &score.CreatedAt, &score.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScoreNotFound
		}
		return nil, err
	}

	return &score, nil
}

func (r *PostgresScoreRepository) GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*model.Score, error) {
	query := `SELECT id, user_id, score, category, difficulty, created_at, deleted_at FROM scores WHERE id = $1`

	var score model.Score
	err := r.db.QueryRow(ctx, query, id).Scan(&score.ID, &score.UserID, &score.Score, &score.Category, &score.Difficulty, &score.CreatedAt, &score.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScoreNotFound
		}
		return nil, err
	}

	return &score, nil
}

func (r *PostgresScoreRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]*model.Score, error) {
	query := `SELECT id, user_id, score, category, difficulty, created_at, deleted_at FROM scores WHERE deleted_at IS NULL AND user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query GetByUser failed: %w", err)
	}
	defer rows.Close()

	scores := make([]*model.Score, 0)
	for rows.Next() {
		var u model.Score
		if err := rows.Scan(&u.ID, &u.UserID, &u.Score, &u.Category, &u.Difficulty, &u.CreatedAt, &u.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan GetByUser row failed: %w", err)
		}
		scores = append(scores, &u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return scores, nil
}

func (r *PostgresScoreRepository) Create(ctx context.Context, score *model.Score) error {
	query := `INSERT INTO scores (user_id, score, category, difficulty) VALUES ($1, $2, $3, $4) RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query, score.UserID, score.Score, score.Category, score.Difficulty).Scan(&score.ID, &score.CreatedAt)
	if err != nil {
		return fmt.Errorf("create score failed: %w", err)
	}

	return nil
}

func (r *PostgresScoreRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE scores SET deleted_at = CURRENT_TIMESTAMP where id = $1 AND deleted_at IS NULL`

	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete score failed: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrScoreNotFound
	}

	return nil
}

func (r *PostgresScoreRepository) Restore(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE scores SET deleted_at = NULL WHERE id = $1 AND deleted_at IS NOT NULL`

	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("restore score failed: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrScoreNotFound
	}

	return nil
}

func (r *PostgresScoreRepository) PermanentDelete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM scores WHERE id = $1`

	commandTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("permanent delete score failed: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return ErrScoreNotFound
	}

	return nil
}

// Helper for leaderboard sync
func (r *PostgresScoreRepository) GetHighestActiveScore(ctx context.Context, userID uuid.UUID, category string) (float64, bool, error) {
	const query = `
		SELECT score
		FROM scores
		WHERE user_id = $1
		  AND category = $2
		  AND deleted_at IS NULL
		ORDER BY score DESC
		LIMIT 1
	`

	var score float64
	err := r.db.QueryRow(ctx, query, userID, category).Scan(&score)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}

	return score, true, nil
}
