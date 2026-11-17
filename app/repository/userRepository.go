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
	ErrUserNotFound = errors.New("user not found")
)

type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUserRepository(db *pgxpool.Pool) model.UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) GetAll(ctx context.Context, filter model.UserFilter) ([]*model.User, int, error) {
	var conditions []string
	args := pgx.NamedArgs{
		"limit":  filter.Limit,
		"offset": filter.Offset,
	}

	// Trashed filter
	switch filter.Trashed {
	case "only":
		conditions = append(conditions, "deleted_at IS NOT NULL")
	case "with":
		// Include both active and soft-deleted users
	default:
		conditions = append(conditions, "deleted_at IS NULL")
	}

	// Search filter
	if filter.Search != "" {
		conditions = append(conditions, `(
			id::text ILIKE @search OR 
			name ILIKE @search OR 
			username ILIKE @search OR 
			email ILIKE @search
		)`)
		args["search"] = "%" + filter.Search + "%"
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT 
			id, name, username, email, created_at, updated_at, deleted_at,
			COUNT(*) OVER() AS total_count
		FROM users
		%s
		ORDER BY created_at DESC, id DESC
		LIMIT @limit OFFSET @offset
	`, whereClause)

	rows, err := r.db.Query(ctx, query, args)
	if err != nil {
		return nil, 0, fmt.Errorf("query GetAll failed: %w", err)
	}

	total := 0
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*model.User, error) {
		var u model.User
		err := row.Scan(
			&u.ID,
			&u.Name,
			&u.Username,
			&u.Email,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.DeletedAt,
			&total,
		)
		return &u, err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("collect GetAll rows failed: %w", err)
	}

	// Fallback count if offset is past the last page (so total stays accurate)
	if len(users) == 0 && filter.Offset > 0 {
		countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM users %s`, whereClause)
		if err := r.db.QueryRow(ctx, countQuery, args).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("fallback count GetAll failed: %w", err)
		}
	}

	return users, total, nil
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

	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = id.String()
	}

	query := `
		SELECT id, username 
		FROM users 
		WHERE id = ANY($1) AND deleted_at IS NULL
	`

	rows, err := r.db.Query(ctx, query, strIDs)
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

	if tag.RowsAffected() == 0 {
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

	if tag.RowsAffected() == 0 {
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
