package service

import (
	"context"
	"fmt"
	"log"

	"github.com/AlifRJ/Go_Auth/app/model"
	"github.com/redis/go-redis/v9"
)

type LeaderboardUser struct {
	UserID string  `json:"user_id"`
	Username string `json:"username"`
	Score  float64 `json:"score"`
	Rank   int64   `json:"rank"` // 1-based rank position
}

type LeaderboardService struct {
	rdb *redis.Client
	userRepo model.UserRepository
}

func NewLeaderboardService(rdb *redis.Client, userRepo model.UserRepository) *LeaderboardService {
	return &LeaderboardService{rdb: rdb, userRepo: userRepo}
}

// Redis Key Formatter
func (s *LeaderboardService) leaderboardKey(category string) string {
	return fmt.Sprintf("leaderboard:%s", category)
}

// SubmitScore updates a user's score in the leaderboard ZSET.
// Uses ZAdd to update or set the absolute score.
func (s *LeaderboardService) SubmitScore(ctx context.Context, category string, userID uint, score float64) error {
	key := s.leaderboardKey(category)
	member := fmt.Sprintf("%d", userID)

	// ZAdd adds or updates score for the member
	err := s.rdb.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to submit score to leaderboard: %w", err)
	}
	return nil
}

// GetTopScores retrieves top N players ordered from highest to lowest score (O(log(N) + M)).
func (s *LeaderboardService) GetTopScores(ctx context.Context, category string, limit int64) ([]LeaderboardUser, error) {
	key := s.leaderboardKey(category)

	// Fetch top scores from Redis ZSET
	results, err := s.rdb.ZRevRangeWithScores(ctx, key, 0, limit-1).Result()
	if err != nil {
		log.Printf("Redis leaderboard error: %v. Returning empty or fallback.", err)
		return []LeaderboardUser{}, nil
	}

	// Extract unique User IDs from ZSET members
    userIDs := make([]uint, 0, len(results))
	for _, z := range results {
		var id uint
		if _, err := fmt.Sscanf(fmt.Sprintf("%v", z.Member), "%d", &id); err == nil {
			userIDs = append(userIDs, id)
		}
	}

	// Batch query PostgreSQL for usernames in 1 database call
    userMap, err := s.userRepo.GetMapByIDs(ctx, userIDs)
	if err != nil {
		log.Printf("Failed to fetch user map from DB: %v", err)
	}

	leaderboard := make([]LeaderboardUser, 0, len(results))
	for rank, z := range results {
		userIDStr := fmt.Sprintf("%v", z.Member)

		var id uint
		fmt.Sscanf(userIDStr, "%d", &id)

		username := userMap[id]
		if username == "" {
			username = fmt.Sprintf("Player %s", userIDStr) // Fallback if user was deleted or missing
		}

		leaderboard = append(leaderboard, LeaderboardUser{
			UserID:   userIDStr,
			Username: username,
			Score:    z.Score,
			Rank:     int64(rank + 1),
		})
	}

	// leaderboard := make([]LeaderboardUser, 0, len(results))
	// for rank, z := range results {
	// 	leaderboard = append(leaderboard, LeaderboardUser{
	// 		UserID: fmt.Sprintf("%v", z.Member),
	// 		Score:  z.Score,
	// 		Rank:   int64(rank + 1), // 1-based rank index
	// 	})
	// }

	return leaderboard, nil
}

// GetUserRank gets a specific user's position and score in the leaderboard.
func (s *LeaderboardService) GetUserRank(ctx context.Context, category string, userID uint) (*LeaderboardUser, error) {
	key := s.leaderboardKey(category)
	member := fmt.Sprintf("%d", userID)

	// ZRevRank returns 0-based rank index (0 = 1st place)
	rank, err := s.rdb.ZRevRank(ctx, key, member).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // User not found in leaderboard
		}
		return nil, fmt.Errorf("failed to fetch user rank: %w", err)
	}

	// Fetch actual score
	score, err := s.rdb.ZScore(ctx, key, member).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user score: %w", err)
	}

	return &LeaderboardUser{
		UserID: member,
		Score:  score,
		Rank:   rank + 1, // Convert 0-based to 1-based rank
	}, nil
}

// RemoveUser removes a user from a specific leaderboard ZSET
func (s *LeaderboardService) RemoveUser(ctx context.Context, category string, userID uint) error {
	key := s.leaderboardKey(category)
	member := fmt.Sprintf("%d", userID)
	return s.rdb.ZRem(ctx, key, member).Err()
}