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
	ErrUserNotFound = errors.New("user not found")
)

type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUserRepository(db *pgxpool.Pool) model.UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) GetAll(ctx context.Context, limit, offset int) ([]*model.User, error) {
	query := `SELECT id, name, username, email, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL ORDER BY id DESC LIMIT $1 OFFSET $2`
	
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query GetAll failed: %w", err)
	} 
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan GetAll row failed: %w", err)
		}
		users = append(users, &u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return users, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `SELECT id, name, username, email, created_at, updated_at, deleted_at FROM users WHERE id = $1 AND deleted_at IS NULL`
	
	var u model.User
	err := r.db.QueryRow(ctx, query, id).Scan(&u.ID, &u.Name, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	
	return &u, nil
}

func (r *PostgresUserRepository) GetMapByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]string), nil
	}

	query := `
		SELECT id, username 
		FROM users 
		WHERE id = ANY($1) AND deleted_at IS NULL
	`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userMap := make(map[uuid.UUID]string, len(ids))
	for rows.Next() {
		var id uuid.UUID
		var username string
		if err := rows.Scan(&id, &username); err != nil {
			return nil, err
		}
		userMap[id] = username
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userMap, nil
}

func (r *PostgresUserRepository) GetByEmailOrUsername(ctx context.Context, identifier string) (*model.User, error) {
	query := `
		SELECT id, name, username, email, password, created_at, updated_at, deleted_at 
		FROM users 
		WHERE (email = $1 OR username = $1) AND deleted_at IS NULL`

	var u model.User
	err := r.db.QueryRow(ctx, query, identifier).Scan(
		&u.ID, &u.Name, &u.Username, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, name, username, email, created_at, updated_at, deleted_at 
		FROM users 
		WHERE email = $1 AND deleted_at IS NULL`

	var u model.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Name, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

func (r *PostgresUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, name, username, email, created_at, updated_at, deleted_at 
		FROM users 
		WHERE username = $1 AND deleted_at IS NULL`

	var u model.User
	err := r.db.QueryRow(ctx, query, username).Scan(
		&u.ID, &u.Name, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (name, username, email, password) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	
	err := r.db.QueryRow(ctx, query, user.Name, user.Username, user.Email, user.Password).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return fmt.Errorf("create user failed: %w", err)
	}
	
	return nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *model.User) error {
	query := `UPDATE users SET name=$1, username=$2, email=$3, password=$4, updated_at=CURRENT_TIMESTAMP WHERE id = $5 AND deleted_at IS NULL`
	
	tag, err := r.db.Exec(ctx, query, user.Name, user.Username, user.Email, user.Password, user.ID)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0{
		return ErrUserNotFound
	}
	
	return nil
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET deleted_at = CURRENT_TIMESTAMP where id = $1 AND deleted_at IS NULL`
	
	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0{
		return ErrUserNotFound
	}
	
	return nil
}

func (r *PostgresUserRepository) PermanentDelete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	
	commandTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	
	if commandTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	
	return nil
}
