package model

import (
	"context"
	"time"
)

type User struct {
	ID         uint			`json:"id" db:"id"`
	Name       string		`json:"name" db:"name"`
	Username   string		`json:"username" db:"username"`
	Email      string		`json:"email" db:"email"`
	Password   string		`json:"-" db:"password"`
	CreatedAt time.Time	`json:"created_at" db:"created_at"`
	UpdatedAt *time.Time	`json:"updated_at,omitempty" db:"updated_at"`
	DeletedAt *time.Time	`json:"deleted_at,omitempty" db:"deleted_at"`
}

type UserRepository interface {
	GetAll(ctx context.Context, limit, offset int) ([]*User, error)
	GetByID(ctx context.Context, id uint) (*User, error)
	GetMapByIDs(ctx context.Context, ids []uint) (map[uint]string, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmailOrUsername(ctx context.Context, identifier string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
	PermanentDelete(ctx context.Context, id uint) error
}