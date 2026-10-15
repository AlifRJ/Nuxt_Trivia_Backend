package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"uuid"

	"github.com/AlifRJ/Go_Auth/app/model"
	"github.com/redis/go-redis/v9"
)

type cachedScoreRepository struct {
	rdb      *redis.Client
	dbRepo   model.ScoreRepository
	cacheTTL time.Duration
}

func NewCachedScoreRepository(rdb *redis.Client, dbRepo model.ScoreRepository, ttl time.Duration) model.ScoreRepository {
	return &cachedScoreRepository{
		rdb:      rdb,
		dbRepo:   dbRepo,
		cacheTTL: ttl,
	}
}

// Key formatters
func (c *cachedScoreRepository) scoresAllKey(limit, offset int) string {
	return fmt.Sprintf("scores:all:limit:%d:offset:%d", limit, offset)
}

func (c *cachedScoreRepository) userScoresKey(userID uuid.UUID) string {
	return fmt.Sprintf("scores:user:%s", userID.String())
}

// func (c *cachedScoreRepository) userScoresPattern(userID uint) string {
// 	return fmt.Sprintf("scores:user:%d*", userID)
// }

// Clear all paginated GetAll caches
func (c *cachedScoreRepository) invalidateAllScoresCache(ctx context.Context) {
	iter := c.rdb.Scan(ctx, 0, "scores:all:*", 0).Iterator()
	for iter.Next(ctx) {
		_ = c.rdb.Del(ctx, iter.Val()).Err()
	}
}

func (c *cachedScoreRepository) GetAll(ctx context.Context, limit, offset int) ([]*model.Score, error) {
	key := c.scoresAllKey(limit, offset)

	// Fetch from Redis
	val, err := c.rdb.Get(ctx, key).Result()
	if err == nil {
		var scores []*model.Score
		if err := json.Unmarshal([]byte(val), &scores); err == nil {
			return scores, nil // Cache Hit
		}
	}

	// Cache Miss: Fetch from database
	scores, err := c.dbRepo.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Save to Redis
	if data, err := json.Marshal(scores); err == nil {
		_ = c.rdb.Set(ctx, key, data, c.cacheTTL).Err()
	}

	return scores, nil
}

func (c *cachedScoreRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]*model.Score, error) {
	key := c.userScoresKey(userID)

	// Fetch from Redis
	val, err := c.rdb.Get(ctx, key).Result()
	if err == nil {
		var scores []*model.Score
		if err := json.Unmarshal([]byte(val), &scores); err == nil {
			return scores, nil // Cache Hit
		}
	}

	// Cache Miss: Fetch from database
	scores, err := c.dbRepo.GetByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Save to Redis
	if data, err := json.Marshal(scores); err == nil {
		_ = c.rdb.Set(ctx, key, data, c.cacheTTL).Err()
	}

	return scores, nil
}

func (c *cachedScoreRepository) GetByID(ctx context.Context, userID uuid.UUID) (*model.Score, error) {
	key := c.userScoresKey(userID)

	// Fetch From Redis
	val, err := c.rdb.Get(ctx, key).Result()
	if err == nil {
		var score model.Score
		if err := json.Unmarshal([]byte(val), &score); err == nil {
			return &score, nil // Cache Hit
		}
	}

	// Cache Miss: Fetch From Database
	score, err := c.dbRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Save to Redis
	if data, err := json.Marshal(score); err == nil {
		_ = c.rdb.Set(ctx, key, data, c.cacheTTL).Err()
	}

	return score, nil
}

func (c *cachedScoreRepository) Create(ctx context.Context, score *model.Score) error {
	if err := c.dbRepo.Create(ctx, score); err != nil {
		return err
	}

	// Invalidate cache for the specific user and overall list
	_ = c.rdb.Del(ctx, c.userScoresKey(score.UserID)).Err()
	c.invalidateAllScoresCache(ctx)

	return nil
}

func (c *cachedScoreRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.dbRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cached lists across DB and user caches
	c.invalidateAllScoresCache(ctx)
	c.invalidateUserScoresPattern(ctx)

	return nil
}

func (c *cachedScoreRepository) PermanentDelete(ctx context.Context, id uuid.UUID) error {
	if err := c.dbRepo.PermanentDelete(ctx, id); err != nil {
		return err
	}

	// Invalidate cached lists across DB and user caches
	c.invalidateAllScoresCache(ctx)
	c.invalidateUserScoresPattern(ctx)

	return nil
}

// Invalidate user score keys when a score ID is mutated without knowing UserID
func (c *cachedScoreRepository) invalidateUserScoresPattern(ctx context.Context) {
	iter := c.rdb.Scan(ctx, 0, "scores:user:*", 0).Iterator()
	for iter.Next(ctx) {
		_ = c.rdb.Del(ctx, iter.Val()).Err()
	}
}