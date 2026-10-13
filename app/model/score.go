package model

import (
	"context"
	"time"
)

type Score struct {
	ID         	uint       	`json:"id" db:"id"`
	UserID     	uint       	`json:"user_id" db:"user_id"`
	Score      	uint       	`json:"score" db:"score"`
	Category   	string     	`json:"category" db:"category"`
	Difficulty 	string     	`json:"difficulty" db:"difficulty"`
	CreatedAt  	time.Time  	`json:"created_at" db:"created_at"`
	DeletedAt  	*time.Time 	`json:"deleted_at,omitempty" db:"deleted_at"`
}

type ScoreRepository interface{
	GetAll(ctx context.Context, limit, offset int) ([]*Score, error)
	GetByUser(ctx context.Context, userID uint) ([]*Score, error)
	Create(ctx context.Context, score *Score) error
	Delete(ctx context.Context, id uint) error
	PermanentDelete(ctx context.Context, id uint) error
}
