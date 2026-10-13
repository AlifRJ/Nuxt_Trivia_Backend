package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AlifRJ/Go_Auth/app/model"
	"github.com/redis/go-redis/v9"
)

type cachedAuthRepository struct {
	rdb *redis.Client
	authRepo  model.AuthRepository
}

func NewCachedAuthRepository(rdb *redis.Client, authRepo model.AuthRepository) model.AuthRepository {
	return &cachedAuthRepository{
		rdb: rdb,
		authRepo: authRepo,
	}
}

// Format Key Helpers
func (r *cachedAuthRepository) refreshTokenKey(userID uint) string {
	return fmt.Sprintf("auth:refresh_token:%d", userID)
}

func (r *cachedAuthRepository) blacklistKey(jti string) string {
	return fmt.Sprintf("auth:blacklist:%s", jti)
}

// Store Refresh Token
func (r *cachedAuthRepository) StoreRefreshToken(ctx context.Context, userID uint, token string, ttl time.Duration) error {
	// Save to Database
	if err := r.authRepo.StoreRefreshToken(ctx, userID, token, ttl); err != nil {
		return err
	}

	// Save to Cache
	key := r.refreshTokenKey(userID)
	if err := r.rdb.Set(ctx, key, token, ttl).Err(); err != nil {
		return fmt.Errorf("failed to cache refresh token: %w", err)
	}

	return nil
}

// Verify Refresh Token
func (r *cachedAuthRepository) VerifyRefreshToken(ctx context.Context, userID uint, token string) (bool, error) {
	key := r.refreshTokenKey(userID)

	// Check in Cache
	storedToken, err := r.rdb.Get(ctx, key).Result()
	if err == nil {
		return storedToken == token, nil
	}

	// Fallback to Database if Cache Miss
	if errors.Is(err, redis.Nil) {
		valid, err := r.authRepo.VerifyRefreshToken(ctx, userID, token)
		if err != nil || !valid {
			return false, err
		}

		// Cache Warm-up (7 Days)
		_ = r.rdb.Set(ctx, key, token, 7*24*time.Hour).Err()
		return true, nil
	}

	return false, fmt.Errorf("redis error on verify refresh token: %w", err)
}

// Delete Refresh Token
func (r *cachedAuthRepository) DeleteRefreshToken(ctx context.Context, userID uint) error {
	// Delete from Database
	_ = r.authRepo.DeleteRefreshToken(ctx, userID)

	// Delete from Cache
	key := r.refreshTokenKey(userID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete refresh token from cache: %w", err)
	}

	return nil
}

// Blacklist Access Token
func (r *cachedAuthRepository) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	key := r.blacklistKey(jti)
	if err := r.rdb.Set(ctx, key, "revoked", ttl).Err(); err != nil {
		return fmt.Errorf("failed to blacklist access token: %w", err)
	}
	return nil
}

// Check if Token is Blacklisted
func (r *cachedAuthRepository) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := r.blacklistKey(jti)
	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis error on blacklist lookup: %w", err)
	}
	return exists > 0, nil
}
