package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

type cachedScoresResult struct {
	Scores []*model.ScoreWithUsername `json:"scores"`
	Total  int                        `json:"total"`
}

func NewCachedScoreRepository(rdb *redis.Client, dbRepo model.ScoreRepository, ttl time.Duration) model.ScoreRepository {
	return &cachedScoreRepository{
		rdb:      rdb,
		dbRepo:   dbRepo,
		cacheTTL: ttl,
	}
}

// Key formatters
func (c *cachedScoreRepository) scoresAllKey(filter model.ScoreFilter) string {
	// Normalize Search
	search := strings.ToLower(strings.TrimSpace(filter.Search))

	// Normalize empty Trashed so "" and "without" don't create duplicate cache entries
	trashed := strings.ToLower(strings.TrimSpace(filter.Trashed))
	if trashed == "" {
		trashed = "default"
	}

	return fmt.Sprintf(
		"scores:all:trashed:%s:search:%q:limit:%d:offset:%d",
		trashed,
		search,
		filter.Limit,
		filter.Offset,
	)
}

func (c *cachedScoreRepository) userScoresKey(userID uuid.UUID) string {
	return fmt.Sprintf("scores:user:%s", userID.String())
}

// Clear all paginated GetAll caches
func (c *cachedScoreRepository) invalidateAllScoresCache(ctx context.Context) {
	iter := c.rdb.Scan(ctx, 0, "scores:all:*", 0).Iterator()
	for iter.Next(ctx) {
		_ = c.rdb.Del(ctx, iter.Val()).Err()
	}
}

func (c *cachedScoreRepository) GetAll(ctx context.Context, filter model.ScoreFilter) ([]*model.ScoreWithUsername, int, error) {
	key := c.scoresAllKey(filter)

	// Fetch from Redis
	val, err := c.rdb.Get(ctx, key).Result()
	if err == nil {
		var cached cachedScoresResult
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			return cached.Scores, cached.Total, nil // Cache Hit
		}
	}

	// Cache Miss: Fetch from database
	scores, total, err := c.dbRepo.GetAll(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Save to Redis
	payload := cachedScoresResult{
		Scores: scores,
		Total:  total,
	}
	if data, err := json.Marshal(payload); err == nil {
		_ = c.rdb.Set(ctx, key, data, c.cacheTTL).Err()
	}

	return scores, total, nil
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
