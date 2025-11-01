package service

import (
	"ariand/internal/db/sqlc"
	"ariand/internal/types"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/type/money"
)

type AccountSummary struct {
	Summary *sqlc.GetDashboardSummaryForAccountRow
	Trends  []sqlc.GetDashboardTrendsForAccountRow
}

type PeriodType int

const (
	Period7Days PeriodType = iota
	Period30Days
	Period90Days
	PeriodCustom
)

type CategorySpendingParams struct {
	UserID      uuid.UUID
	PeriodType  PeriodType
	CustomStart *time.Time
	CustomEnd   *time.Time
	Timezone    string
}

type PeriodInfo struct {
	StartDate string
	EndDate   string
	Label     string
}

type CategorySpendingResult struct {
	CurrentPeriod  PeriodInfo
	PreviousPeriod PeriodInfo
	Current        []sqlc.GetCategorySpendingForPeriodRow
	Previous       []sqlc.GetCategorySpendingForPeriodRow
}

type DashboardService interface {
	Balance(ctx context.Context, userID uuid.UUID) (*money.Money, error)
	Debt(ctx context.Context, userID uuid.UUID) (*money.Money, error)
	NetBalance(ctx context.Context, userID uuid.UUID) (*money.Money, error)
	Trends(ctx context.Context, params sqlc.GetDashboardTrendsParams) ([]sqlc.GetDashboardTrendsRow, error)
	Summary(ctx context.Context, params sqlc.GetDashboardSummaryParams) (*sqlc.GetDashboardSummaryRow, error)
	MonthlyComparison(ctx context.Context, params sqlc.GetMonthlyComparisonParams) ([]sqlc.GetMonthlyComparisonRow, error)
	TopCategories(ctx context.Context, params sqlc.GetTopCategoriesParams) ([]sqlc.GetTopCategoriesRow, error)
	TopMerchants(ctx context.Context, params sqlc.GetTopMerchantsParams) ([]sqlc.GetTopMerchantsRow, error)
	AccountBalances(ctx context.Context, userID uuid.UUID) ([]sqlc.GetAccountBalancesRow, error)
	GetAccountSummary(ctx context.Context, userID uuid.UUID, accountID int64, startDate *string, endDate *string) (*AccountSummary, error)
	GetSpendingTrends(ctx context.Context, userID uuid.UUID, startDate string, endDate string, categoryID *int64, accountID *int64) ([]sqlc.GetDashboardTrendsRow, error)
	GetCategorySpendingComparison(ctx context.Context, params CategorySpendingParams) (*CategorySpendingResult, error)
}

type dashSvc struct {
	queries *sqlc.Queries
}

func newDashSvc(queries *sqlc.Queries) DashboardService {
	return &dashSvc{queries: queries}
}

func (s *dashSvc) Balance(ctx context.Context, userID uuid.UUID) (*money.Money, error) {
	balances, err := s.AccountBalances(ctx, userID)
	if err != nil {
		return nil, wrapErr("DashboardService.Balance", err)
	}

	total := money.Money{CurrencyCode: "CAD", Units: 0, Nanos: 0}

	for _, balance := range balances {
		hasBalance := len(balance.CurrentBalance) > 0
		if !hasBalance {
			continue
		}

		var m types.Money
		if err := m.Scan(balance.CurrentBalance); err != nil {
			continue
		}

		if types.IsPositive(&m.Money) {
			total, err = types.AddMoney(&total, &m.Money)
			if err != nil {
				return nil, wrapErr("DashboardService.Balance.AddMoney", err)
			}
		}
	}

	return &total, nil
}

func (s *dashSvc) Debt(ctx context.Context, userID uuid.UUID) (*money.Money, error) {
	balances, err := s.AccountBalances(ctx, userID)
	if err != nil {
		return nil, wrapErr("DashboardService.Debt", err)
	}

	total := money.Money{CurrencyCode: "CAD", Units: 0, Nanos: 0}

	for _, balance := range balances {
		hasBalance := len(balance.CurrentBalance) > 0
		if !hasBalance {
			continue
		}

		var m types.Money
		if err := m.Scan(balance.CurrentBalance); err != nil {
			continue
		}

		if types.IsNegative(&m.Money) {
			absoluteAmount := types.Negate(&m.Money)
			total, err = types.AddMoney(&total, &absoluteAmount)
			if err != nil {
				return nil, wrapErr("DashboardService.Debt.AddMoney", err)
			}
		}
	}

	return &total, nil
}

func (s *dashSvc) Trends(ctx context.Context, params sqlc.GetDashboardTrendsParams) ([]sqlc.GetDashboardTrendsRow, error) {
	trends, err := s.queries.GetDashboardTrends(ctx, params)
	if err != nil {
		return nil, wrapErr("DashboardService.Trends", err)
	}
	return trends, nil
}

func (s *dashSvc) Summary(ctx context.Context, params sqlc.GetDashboardSummaryParams) (*sqlc.GetDashboardSummaryRow, error) {
	summary, err := s.queries.GetDashboardSummary(ctx, params)
	if err != nil {
		return nil, wrapErr("DashboardService.Summary", err)
	}
	return &summary, nil
}

func (s *dashSvc) MonthlyComparison(ctx context.Context, params sqlc.GetMonthlyComparisonParams) ([]sqlc.GetMonthlyComparisonRow, error) {
	comparison, err := s.queries.GetMonthlyComparison(ctx, params)
	if err != nil {
		return nil, wrapErr("DashboardService.MonthlyComparison", err)
	}
	return comparison, nil
}

func (s *dashSvc) TopCategories(ctx context.Context, params sqlc.GetTopCategoriesParams) ([]sqlc.GetTopCategoriesRow, error) {
	categories, err := s.queries.GetTopCategories(ctx, params)
	if err != nil {
		return nil, wrapErr("DashboardService.TopCategories", err)
	}
	return categories, nil
}

func (s *dashSvc) TopMerchants(ctx context.Context, params sqlc.GetTopMerchantsParams) ([]sqlc.GetTopMerchantsRow, error) {
	merchants, err := s.queries.GetTopMerchants(ctx, params)
	if err != nil {
		return nil, wrapErr("DashboardService.TopMerchants", err)
	}
	return merchants, nil
}

func (s *dashSvc) NetBalance(ctx context.Context, userID uuid.UUID) (*money.Money, error) {
	balances, err := s.AccountBalances(ctx, userID)
	if err != nil {
		return nil, wrapErr("DashboardService.NetBalance", err)
	}

	total := money.Money{CurrencyCode: "CAD", Units: 0, Nanos: 0}

	for _, balance := range balances {
		hasBalance := len(balance.CurrentBalance) > 0
		if !hasBalance {
			continue
		}

		var m types.Money
		if err := m.Scan(balance.CurrentBalance); err != nil {
			continue
		}

		total, err = types.AddMoney(&total, &m.Money)
		if err != nil {
			return nil, wrapErr("DashboardService.NetBalance.AddMoney", err)
		}
	}

	return &total, nil
}

func (s *dashSvc) AccountBalances(ctx context.Context, userID uuid.UUID) ([]sqlc.GetAccountBalancesRow, error) {
	balances, err := s.queries.GetAccountBalances(ctx, userID)
	if err != nil {
		return nil, wrapErr("DashboardService.AccountBalances", err)
	}
	return balances, nil
}

func (s *dashSvc) GetAccountSummary(ctx context.Context, userID uuid.UUID, accountID int64, startDate *string, endDate *string) (*AccountSummary, error) {
	var start, end *time.Time

	hasStartDate := startDate != nil
	if hasStartDate {
		parsed, err := time.Parse("2006-01-02", *startDate)
		if err != nil {
			return nil, wrapErr("DashboardService.GetAccountSummary.ParseStartDate", err)
		}
		start = &parsed
	}

	hasEndDate := endDate != nil
	if hasEndDate {
		parsed, err := time.Parse("2006-01-02", *endDate)
		if err != nil {
			return nil, wrapErr("DashboardService.GetAccountSummary.ParseEndDate", err)
		}
		end = &parsed
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
		return nil, wrapErr("DashboardService.GetAccountSummary.GetTrends", err)
	}

	return &AccountSummary{
		Summary: &summary,
		Trends:  trends,
	}, nil
}

func (s *dashSvc) GetSpendingTrends(ctx context.Context, userID uuid.UUID, startDate string, endDate string, categoryID *int64, accountID *int64) ([]sqlc.GetDashboardTrendsRow, error) {
	var start, end *time.Time

	parsedStart, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, wrapErr("DashboardService.GetSpendingTrends.ParseStartDate", err)
	}
	start = &parsedStart

	parsedEnd, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, wrapErr("DashboardService.GetSpendingTrends.ParseEndDate", err)
	}
	end = &parsedEnd

	params := sqlc.GetDashboardTrendsParams{
		UserID: userID,
		Start:  start,
		End:    end,
	}

	// TODO: currently the database query doesn't support filtering by category or account
	// these parameters are included for future extensibility but ignored for now in this MVP
	_ = categoryID
	_ = accountID

	return s.Trends(ctx, params)
}

func (s *dashSvc) GetCategorySpendingComparison(
	ctx context.Context,
	params CategorySpendingParams,
) (*CategorySpendingResult, error) {
	// Calculate period boundaries (timezone support coming later from user preferences)
	loc := time.UTC
	now := time.Now().In(loc)

	var currentStart, currentEnd, previousStart, previousEnd time.Time
	var currentLabel, previousLabel string

	switch params.PeriodType {
	case Period7Days:
		currentStart = startOfDay(now.AddDate(0, 0, -6), loc)
		currentEnd = endOfDay(now, loc)
		previousEnd = startOfDay(currentStart, loc).Add(-time.Nanosecond)
		previousStart = startOfDay(now.AddDate(0, 0, -13), loc)
		currentLabel = "Last 7 Days"
		previousLabel = "Previous 7 Days"

	case Period30Days:
		currentStart = startOfDay(now.AddDate(0, 0, -29), loc)
		currentEnd = endOfDay(now, loc)
		previousEnd = startOfDay(currentStart, loc).Add(-time.Nanosecond)
		previousStart = startOfDay(now.AddDate(0, 0, -59), loc)
		currentLabel = "Last 30 Days"
		previousLabel = "Previous 30 Days"

	case Period90Days:
		currentStart = startOfDay(now.AddDate(0, 0, -89), loc)
		currentEnd = endOfDay(now, loc)
		previousEnd = startOfDay(currentStart, loc).Add(-time.Nanosecond)
		previousStart = startOfDay(now.AddDate(0, 0, -179), loc)
		currentLabel = "Last 90 Days"
		previousLabel = "Previous 90 Days"

	case PeriodCustom:
		if params.CustomStart == nil || params.CustomEnd == nil {
			return nil, wrapErr("DashboardService.GetCategorySpendingComparison",
				fmt.Errorf("custom period requires both start and end dates"))
		}

		currentStart = startOfDay(*params.CustomStart, loc)
		currentEnd = endOfDay(*params.CustomEnd, loc)

		if currentEnd.Before(currentStart) {
			return nil, wrapErr("DashboardService.GetCategorySpendingComparison",
				fmt.Errorf("end date must be after start date"))
		}

		duration := currentEnd.Sub(currentStart) + time.Nanosecond
		previousEnd = startOfDay(currentStart, loc).Add(-time.Nanosecond)
		previousStart = currentStart.Add(-duration)

		currentLabel = formatDateRange(currentStart, currentEnd)
		previousLabel = formatDateRange(previousStart, previousEnd)

	default:
		return nil, wrapErr("DashboardService.GetCategorySpendingComparison",
			fmt.Errorf("invalid period type"))
	}

	// Query current period
	current, err := s.queries.GetCategorySpendingForPeriod(ctx, sqlc.GetCategorySpendingForPeriodParams{
		UserID:    params.UserID,
		StartDate: currentStart,
		EndDate:   currentEnd,
	})
	if err != nil {
		return nil, wrapErr("DashboardService.GetCategorySpendingComparison.Current", err)
	}

	// Query previous period
	previous, err := s.queries.GetCategorySpendingForPeriod(ctx, sqlc.GetCategorySpendingForPeriodParams{
		UserID:    params.UserID,
		StartDate: previousStart,
		EndDate:   previousEnd,
	})
	if err != nil {
		return nil, wrapErr("DashboardService.GetCategorySpendingComparison.Previous", err)
	}

	return &CategorySpendingResult{
		CurrentPeriod: PeriodInfo{
			StartDate: currentStart.Format("2006-01-02"),
			EndDate:   currentEnd.Format("2006-01-02"),
			Label:     currentLabel,
		},
		PreviousPeriod: PeriodInfo{
			StartDate: previousStart.Format("2006-01-02"),
			EndDate:   previousEnd.Format("2006-01-02"),
			Label:     previousLabel,
		},
		Current:  current,
		Previous: previous,
	}, nil
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func endOfDay(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, loc)
}

func formatDateRange(start, end time.Time) string {
	if start.Year() == end.Year() && start.Month() == end.Month() {
		return fmt.Sprintf("%s %d-%d, %d",
			start.Month().String()[:3], start.Day(), end.Day(), start.Year())
	}
	return fmt.Sprintf("%s %d, %d - %s %d, %d",
		start.Month().String()[:3], start.Day(), start.Year(),
		end.Month().String()[:3], end.Day(), end.Year())
}
