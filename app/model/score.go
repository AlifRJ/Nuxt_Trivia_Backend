package model

import (
	"context"
	"time"
	"uuid"
)

type Score struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	Score      uint       `json:"score" db:"score"`
	Category   string     `json:"category" db:"category"`
	Difficulty string     `json:"difficulty" db:"difficulty"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type ScoreWithUsername struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	Username   *string    `json:"username" db:"username"`
	Score      uint       `json:"score" db:"score"`
	Category   string     `json:"category" db:"category"`
	Difficulty string     `json:"difficulty" db:"difficulty"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type ScoreFilter struct {
	Search  string
	Trashed string
	Limit   int
	Offset  int
}

type ScoreRepository interface {
	GetAll(ctx context.Context, filter ScoreFilter) ([]*ScoreWithUsername, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Score, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]*Score, error)
	Create(ctx context.Context, score *Score) error
	Delete(ctx context.Context, id uuid.UUID) error
	PermanentDelete(ctx context.Context, id uuid.UUID) error
}
