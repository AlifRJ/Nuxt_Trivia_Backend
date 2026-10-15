package repository

import (
	"context"
	"errors"
	"fmt"
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

func NewPostgresScoreRepository(db *pgxpool.Pool) model.ScoreRepository{
 	return &PostgresScoreRepository{db: db}
}

func (r *PostgresScoreRepository) GetAll(ctx context.Context, limit, offset int) ([]*model.Score, error) {
	query := `SELECT id, user_id, score, category, difficulty, created_at, deleted_at FROM scores WHERE deleted_at IS NULL ORDER BY id DESC LIMIT $1 OFFSET $2`
	
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query GetAll failed: %w", err)
	} 
	defer rows.Close()

	capacity := limit
	if capacity <= 0 {
		capacity = 0
	}
	scores := make([]*model.Score, 0, capacity)
	
	for rows.Next() {
		var u model.Score
		if err := rows.Scan(&u.ID, &u.UserID, &u.Score, &u.Category, &u.Difficulty, &u.CreatedAt, &u.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan GetAll row failed: %w", err)
		}
		scores = append(scores, &u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return scores, nil
}

func (r *PostgresScoreRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Score, error) {
	query := `SELECT id, user_id, score, category, difficulty, created_at, deleted_at FROM scores WHERE deleted_at IS NULL AND id = $1`

	
	var score model.Score
	err := r.db.QueryRow(ctx, query, id).Scan(&score.ID, &score.UserID, &score.Score, &score.Category, &score.Difficulty, &score.CreatedAt, &score.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
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

	if tag.RowsAffected() == 0{
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