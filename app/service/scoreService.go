package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"uuid"

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

func (s *ScoreService) GetByUser(ctx context.Context, userID uuid.UUID) ([]*model.Score, error) {
	return s.repo.GetByUser(ctx, userID)
}

func (s *ScoreService) RegisterScore(ctx context.Context, userID uuid.UUID, scoreVal uint, category, difficulty string) (*model.Score, error) {
	category = strings.ToLower(strings.TrimSpace(category))
	difficulty = strings.ToLower(strings.TrimSpace(difficulty))
	
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
		return nil, fmt.Errorf("failed to register score: %w", err)
	}

	// Push to Redis ZSET Leaderboard (Global & Category specific)
	if s.leaderboardService != nil {
		_ = s.leaderboardService.SubmitScore(ctx, "global", userID, float64(scoreVal))
		_ = s.leaderboardService.SubmitScore(ctx, category, userID, float64(scoreVal))
	}

	return score, nil
}

func (s *ScoreService) DeleteScore(ctx context.Context, id uuid.UUID) error {
	scoreModel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if s.leaderboardService != nil && scoreModel != nil {
		if err := s.leaderboardService.RemoveUser(ctx, scoreModel.Category, scoreModel.UserID); err != nil {
			log.Printf("[WARN] Failed to remove user %d from leaderboard ZSET: %v", scoreModel.UserID, err)
		}
	}

	return nil
}

func (s *ScoreService) DeleteScorePermanently(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.PermanentDelete(ctx, id); err != nil {
		return err
	}
	return nil
}