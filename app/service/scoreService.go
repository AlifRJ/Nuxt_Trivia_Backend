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
	repo               model.ScoreRepository
	leaderboardService *LeaderboardService
}

func NewScoreService(repo model.ScoreRepository, leaderboardService *LeaderboardService) *ScoreService {
	return &ScoreService{repo: repo, leaderboardService: leaderboardService}
}

func (s *ScoreService) GetAll(ctx context.Context, filter model.ScoreFilter) ([]*model.ScoreWithUsername, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = defaultUserLimit
	} else if filter.Limit > maxUserLimit {
		filter.Limit = maxUserLimit
	}

	if filter.Offset < 0 {
		filter.Offset = 0
	}

	filter.Search = strings.TrimSpace(filter.Search)
	if len(filter.Search) > 100 {
		filter.Search = filter.Search[:100]
	}

	switch strings.ToLower(strings.TrimSpace(filter.Trashed)) {
	case "only", "with":
		filter.Trashed = strings.ToLower(strings.TrimSpace(filter.Trashed))
	default:
		filter.Trashed = ""
	}

	return s.repo.GetAll(ctx, filter)
}

func (s *ScoreService) GetByUser(ctx context.Context, userID uuid.UUID) ([]*model.Score, error) {
	return s.repo.GetByUser(ctx, userID)
}

func (s *ScoreService) RegisterScore(ctx context.Context, userID uuid.UUID, scoreVal uint, category, difficulty string) (*model.Score, error) {
	category = strings.ToLower(strings.TrimSpace(category))
	difficulty = strings.ToLower(strings.TrimSpace(difficulty))

	// Validate empty request
	if category == "" || difficulty == "" {
		return nil, ErrInvalidInput
	}

	score := &model.Score{
		UserID:     userID,
		Score:      scoreVal,
		Category:   category,
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
