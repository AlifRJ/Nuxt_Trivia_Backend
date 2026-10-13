package service

import (
	"context"
	"errors"

	"github.com/AlifRJ/Go_Auth/app/model"
)

var (
	ErrInvalidInput = errors.New("category and difficulty cannot be empty")
)

type ScoreService struct {
	repo model.ScoreRepository
	leaderboardService *LeaderboardService
}

func NewScoreService(repo model.ScoreRepository, leaderboardService *LeaderboardService) *ScoreService{
	return &ScoreService{repo: repo, leaderboardService: leaderboardService}
}

func (s *ScoreService) GetAll(ctx context.Context, limit, offset int) ([]*model.Score, error) {
	if limit <= 0 {
		limit = 10 
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}
	return s.repo.GetAll(ctx, limit, offset)
}

func (s *ScoreService) GetByUser(ctx context.Context, userID uint) ([]*model.Score, error) {
	return s.repo.GetByUser(ctx, userID)
}

func (s *ScoreService) RegisterScore(ctx context.Context, userID, scoreVal uint, category, difficulty string) (*model.Score, error) {
	// Validate empty request
	if  category == "" || difficulty == "" {
		return nil, ErrInvalidInput
	}
	
	score := &model.Score{
		UserID: userID, 
		Score: scoreVal, 
		Category: category, 
		Difficulty: difficulty,
	}
	
	if err := s.repo.Create(ctx, score); err != nil {
		return nil, err
	}

	// Push to Redis ZSET Leaderboard (Global & Category specific)
	if s.leaderboardService != nil {
		_ = s.leaderboardService.SubmitScore(ctx, "global", userID, float64(scoreVal))
		_ = s.leaderboardService.SubmitScore(ctx, category, userID, float64(scoreVal))
	}

	return score, nil
}

func (s *ScoreService) DeleteScore(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

func (s *ScoreService) DeleteScorePermanently(ctx context.Context, id uint) error {
	if err := s.repo.PermanentDelete(ctx, id); err != nil {
		return err
	}
	return nil
}