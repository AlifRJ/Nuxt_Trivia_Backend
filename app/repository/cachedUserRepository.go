package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AlifRJ/Go_Auth/app/model"
	"github.com/redis/go-redis/v9"
)

type cachedUserRepository struct {
	rdb *redis.Client
	dbRepo  model.UserRepository
	cacheTTL   time.Duration
}

func NewCachedUserRepository(rdb *redis.Client, dbRepo model.UserRepository, ttl time.Duration) model.UserRepository {
	return &cachedUserRepository{rdb: rdb, dbRepo:	dbRepo, cacheTTL: ttl}
}

// Key formatters
func (c *cachedUserRepository) userKey(id uint) string {
	return fmt.Sprintf("user:%d", id)
}

func (c *cachedUserRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	key := c.userKey(id)

	// Fetch From Redis
	val, err := c.rdb.Get(ctx, key).Result()
	if err == nil {
		var user model.User
		if err := json.Unmarshal([]byte(val), &user); err == nil {
			return &user, nil // Cache Hit
		}
	}

	// Cache Miss: Fetch From Database
	user, err := c.dbRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Save to Redis
	if data, err := json.Marshal(user); err == nil {
		_ = c.rdb.Set(ctx, key, data, c.cacheTTL).Err()
	}

	return user, nil
}

func (c *cachedUserRepository) GetMapByIDs(ctx context.Context, ids []uint) (map[uint]string, error){
	return c.dbRepo.GetMapByIDs(ctx, ids)
}

func (c *cachedUserRepository) GetByEmailOrUsername(ctx context.Context, identifier string) (*model.User, error) {
	return c.dbRepo.GetByEmailOrUsername(ctx, identifier)
}

func (c *cachedUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return c.dbRepo.GetByEmail(ctx, email)
}

func (c *cachedUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return c.dbRepo.GetByUsername(ctx, username)
}

func (c *cachedUserRepository) GetAll(ctx context.Context, limit, offset int) ([]*model.User, error) {
	return c.dbRepo.GetAll(ctx, limit, offset)
}

func (c *cachedUserRepository) Create(ctx context.Context, user *model.User) error {
	return c.dbRepo.Create(ctx, user)
}

func (c *cachedUserRepository) Update(ctx context.Context, user *model.User) error {
	if err := c.dbRepo.Update(ctx, user); err != nil {
		return err
	}

	// Invalidate Cache
	_ = c.rdb.Del(ctx, c.userKey(user.ID)).Err()
	return nil
}

func (c *cachedUserRepository) Delete(ctx context.Context, id uint) error {
	if err := c.dbRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate Cache
	_ = c.rdb.Del(ctx, c.userKey(id)).Err()
	return nil
}

func (c *cachedUserRepository) PermanentDelete(ctx context.Context, id uint) error {
	if err := c.dbRepo.PermanentDelete(ctx, id); err != nil {
		return err
	}

	// Invalidate Cache
	_ = c.rdb.Del(ctx, c.userKey(id)).Err()
	return nil
}