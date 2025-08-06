package service

import (
	sqlc "ariand/internal/db/sqlc"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountSummary struct {
	Summary interface{}
	Trends  interface{}
}

type DashboardService interface {
	BalanceForUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
	DebtForUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
	NetBalanceForUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
	TrendsForUser(ctx context.Context, params sqlc.GetDashboardTrendsForUserParams) ([]sqlc.GetDashboardTrendsForUserRow, error)
	SummaryForUser(ctx context.Context, params sqlc.GetDashboardSummaryForUserParams) (*sqlc.GetDashboardSummaryForUserRow, error)
	MonthlyComparisonForUser(ctx context.Context, params sqlc.GetMonthlyComparisonForUserParams) ([]sqlc.GetMonthlyComparisonForUserRow, error)
	TopCategoriesForUser(ctx context.Context, params sqlc.GetTopCategoriesForUserParams) ([]sqlc.GetTopCategoriesForUserRow, error)
	TopMerchantsForUser(ctx context.Context, params sqlc.GetTopMerchantsForUserParams) ([]sqlc.GetTopMerchantsForUserRow, error)
	AccountBalancesForUser(ctx context.Context, userID uuid.UUID) ([]sqlc.GetAccountBalancesForUserRow, error)
	GetAccountSummary(ctx context.Context, userID uuid.UUID, accountID int64, startDate *string, endDate *string) (*AccountSummary, error)
	GetSpendingTrends(ctx context.Context, userID uuid.UUID, startDate string, endDate string, categoryID *int64, accountID *int64) ([]sqlc.GetDashboardTrendsForUserRow, error)
}

type dashSvc struct {
	queries *sqlc.Queries
}

func newDashSvc(queries *sqlc.Queries) DashboardService {
	return &dashSvc{queries: queries}
}

func (s *dashSvc) BalanceForUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	balances, err := s.AccountBalancesForUser(ctx, userID)
	if err != nil {
		return decimal.Zero, wrapErr("DashboardService.BalanceForUser", err)
	}

	total := decimal.Zero
	for _, balance := range balances {
		if balance.CurrentBalance > 0 {
			balanceDecimal := decimal.NewFromInt32(balance.CurrentBalance)
			total = total.Add(balanceDecimal)
		}
	}

	return total, nil
}

func (s *dashSvc) DebtForUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	balances, err := s.AccountBalancesForUser(ctx, userID)
	if err != nil {
		return decimal.Zero, wrapErr("DashboardService.DebtForUser", err)
	}

	total := decimal.Zero
	for _, balance := range balances {
		if balance.CurrentBalance < 0 {
			balanceDecimal := decimal.NewFromInt32(-balance.CurrentBalance)
			total = total.Add(balanceDecimal)
		}
	}

	return total, nil
}

func (s *dashSvc) TrendsForUser(ctx context.Context, params sqlc.GetDashboardTrendsForUserParams) ([]sqlc.GetDashboardTrendsForUserRow, error) {
	trends, err := s.queries.GetDashboardTrendsForUser(ctx, params)
	if err != nil {
		return nil, wrapErr("DashboardService.TrendsForUser", err)
	}
	return trends, nil
}

func (s *dashSvc) SummaryForUser(ctx context.Context, params sqlc.GetDashboardSummaryForUserParams) (*sqlc.GetDashboardSummaryForUserRow, error) {
	summary, err := s.queries.GetDashboardSummaryForUser(ctx, params)
	if err != nil {
		return nil, wrapErr("DashboardService.SummaryForUser", err)
	}
	return &summary, nil
}

func (s *dashSvc) MonthlyComparisonForUser(ctx context.Context, params sqlc.GetMonthlyComparisonForUserParams) ([]sqlc.GetMonthlyComparisonForUserRow, error) {
	comparison, err := s.queries.GetMonthlyComparisonForUser(ctx, params)
	if err != nil {
		return nil, wrapErr("DashboardService.MonthlyComparisonForUser", err)
	}
	return comparison, nil
}

func (s *dashSvc) TopCategoriesForUser(ctx context.Context, params sqlc.GetTopCategoriesForUserParams) ([]sqlc.GetTopCategoriesForUserRow, error) {
	categories, err := s.queries.GetTopCategoriesForUser(ctx, params)
	if err != nil {
		return nil, wrapErr("DashboardService.TopCategoriesForUser", err)
	}
	return categories, nil
}

func (s *dashSvc) TopMerchantsForUser(ctx context.Context, params sqlc.GetTopMerchantsForUserParams) ([]sqlc.GetTopMerchantsForUserRow, error) {
	merchants, err := s.queries.GetTopMerchantsForUser(ctx, params)
	if err != nil {
		return nil, wrapErr("DashboardService.TopMerchantsForUser", err)
	}
	return merchants, nil
}

func (s *dashSvc) NetBalanceForUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	balances, err := s.AccountBalancesForUser(ctx, userID)
	if err != nil {
		return decimal.Zero, wrapErr("DashboardService.NetBalanceForUser", err)
	}

	total := decimal.Zero
	for _, balance := range balances {
		balanceDecimal := decimal.NewFromInt32(balance.CurrentBalance)
		total = total.Add(balanceDecimal)
	}

	return total, nil
}

func (s *dashSvc) AccountBalancesForUser(ctx context.Context, userID uuid.UUID) ([]sqlc.GetAccountBalancesForUserRow, error) {
	balances, err := s.queries.GetAccountBalancesForUser(ctx, userID)
	if err != nil {
		return nil, wrapErr("DashboardService.AccountBalancesForUser", err)
	}
	return balances, nil
}

func (s *dashSvc) GetAccountSummary(ctx context.Context, userID uuid.UUID, accountID int64, startDate *string, endDate *string) (*AccountSummary, error) {
	var start, end *time.Time
	if startDate != nil {
		if parsed, err := time.Parse("2006-01-02", *startDate); err == nil {
			start = &parsed
		}
	}
	if endDate != nil {
		if parsed, err := time.Parse("2006-01-02", *endDate); err == nil {
			end = &parsed
		}
	}

	summaryParams := sqlc.GetDashboardSummaryForAccountParams{
		UserID:    userID,
		AccountID: accountID,
		Start:     start,
		End:       end,
	}
	summary, err := s.queries.GetDashboardSummaryForAccount(ctx, summaryParams)
	if err != nil {
		return nil, wrapErr("DashboardService.GetAccountSummary", err)
	}

	trendsParams := sqlc.GetDashboardTrendsForAccountParams{
		UserID:    userID,
		AccountID: accountID,
		Start:     start,
		End:       end,
	}
	trends, err := s.queries.GetDashboardTrendsForAccount(ctx, trendsParams)
	if err != nil {
		return nil, wrapErr("DashboardService.GetAccountSummary", err)
	}

	return &AccountSummary{
		Summary: &summary,
		Trends:  trends,
	}, nil
}

func (s *dashSvc) GetSpendingTrends(ctx context.Context, userID uuid.UUID, startDate string, endDate string, categoryID *int64, accountID *int64) ([]sqlc.GetDashboardTrendsForUserRow, error) {
	var start, end *time.Time
	if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
		start = &parsed
	}
	if parsed, err := time.Parse("2006-01-02", endDate); err == nil {
		end = &parsed
	}

	params := sqlc.GetDashboardTrendsForUserParams{
		UserID: userID,
		Start:  start,
		End:    end,
	}

	return s.TrendsForUser(ctx, params)
}
