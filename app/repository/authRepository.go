package repository

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/AlifRJ/Go_Auth/app/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAuthRepository struct {
	db *pgxpool.Pool
}

func NewPostgresAuthRepository(db *pgxpool.Pool) model.AuthRepository {
	return &PostgresAuthRepository{db: db}
}

// Store Refresh Token
func (r *PostgresAuthRepository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)

	query := `
		INSERT INTO auth_sessions (user_id, refresh_token, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) 
		DO UPDATE SET refresh_token = EXCLUDED.refresh_token, expires_at = EXCLUDED.expires_at, updated_at = CURRENT_TIMESTAMP`

	_, err := r.db.Exec(ctx, query, userID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// Verify Refresh Token
func (r *PostgresAuthRepository) VerifyRefreshToken(ctx context.Context, userID uuid.UUID, token string) (bool, error) {
	query := `
		SELECT refresh_token, expires_at 
		FROM auth_sessions 
		WHERE user_id = $1`

	var storedToken string
	var expiresAt time.Time

	err := r.db.QueryRow(ctx, query, userID).Scan(&storedToken, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	// Check if Token Match and not Expired
	if storedToken != token || time.Now().After(expiresAt) {
		return false, nil
	}

	return true, nil
}

// Delete Refresh Token
func (r *PostgresAuthRepository) DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM auth_sessions WHERE user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	return nil
}

func (r *PostgresAuthRepository) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	return nil
}

func (r *PostgresAuthRepository) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	return false, nil
}
