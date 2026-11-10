package service

import (
	"context"
	"fmt"
	"time"

	"github.com/AlifRJ/Go_Auth/app/repository"
	"golang.org/x/sync/errgroup"
)

type AnalyticsService interface {
	GetDashboardAnalytics(ctx context.Context) (*DashboardAnalyticsPayload, error)
}

type analyticsService struct {
	repo repository.AnalyticsRepository
}

func NewAnalyticsService(repo repository.AnalyticsRepository) AnalyticsService {
	return &analyticsService{repo: repo}
}

// Payload structs exported for HTTP handler / JSON response
type ExecutiveKPIs struct {
	TotalGames   int64   `json:"total_games"`
	TotalPlayers int64   `json:"total_players"`
	AvgScore     float64 `json:"avg_score"`
}

type DashboardAnalyticsPayload struct {
	KPIs         ExecutiveKPIs               `json:"kpis"`
	Categories   []repository.CategoryStat   `json:"categories"`
	Difficulties []repository.DifficultyStat `json:"difficulties"`
	DailyTrends  []repository.DailyTrendStat `json:"daily_trends"`
}

func (s *analyticsService) GetDashboardAnalytics(ctx context.Context) (*DashboardAnalyticsPayload, error) {
	// Set execution deadline for all queries combined
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	var (
		totalGames   int64
		totalPlayers int64
		avgScore     float64
		categories   []repository.CategoryStat
		difficulties []repository.DifficultyStat
		dailyTrends  []repository.DailyTrendStat
	)

	// 1. Concurrent Fetch: Executive KPIs
	g.Go(func() error {
		var err error
		totalGames, totalPlayers, avgScore, err = s.repo.GetExecutiveKPIs(ctx)
		if err != nil {
			return fmt.Errorf("kpis: %w", err)
		}
		return nil
	})

	// 2. Concurrent Fetch: Top 5 Categories
	g.Go(func() error {
		var err error
		categories, err = s.repo.GetCategoryStats(ctx, 5)
		if err != nil {
			return fmt.Errorf("categories: %w", err)
		}
		return nil
	})

	// 3. Concurrent Fetch: Difficulty breakdown
	g.Go(func() error {
		var err error
		difficulties, err = s.repo.GetDifficultyStats(ctx)
		if err != nil {
			return fmt.Errorf("difficulties: %w", err)
		}
		return nil
	})

	// 4. Concurrent Fetch: Daily Trends for past 7 days
	g.Go(func() error {
		var err error
		dailyTrends, err = s.repo.GetDailyTrends(ctx, 7)
		if err != nil {
			return fmt.Errorf("daily_trends: %w", err)
		}
		return nil
	})

	// Wait for all 4 queries to finish or fail
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("service: %w", err)
	}

	return &DashboardAnalyticsPayload{
		KPIs: ExecutiveKPIs{
			TotalGames:   totalGames,
			TotalPlayers: totalPlayers,
			AvgScore:     avgScore,
		},
		Categories:   categories,
		Difficulties: difficulties,
		DailyTrends:  dailyTrends,
	}, nil
}
