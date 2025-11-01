package api

import (
	pb "ariand/internal/gen/arian/v1"
	"ariand/internal/service"
	"context"
	"fmt"
	"sort"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/type/date"
)

func (s *Server) GetDashboardSummary(ctx context.Context, req *connect.Request[pb.GetDashboardSummaryRequest]) (*connect.Response[pb.GetDashboardSummaryResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := buildDashboardSummaryParams(userID, req.Msg)
	summary, err := s.services.Dashboard.Summary(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetDashboardSummaryResponse{
		Summary: toProtoDashboardSummary(summary),
	}), nil
}

func (s *Server) GetTrendData(ctx context.Context, req *connect.Request[pb.GetTrendDataRequest]) (*connect.Response[pb.GetTrendDataResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := buildDashboardTrendsParams(userID, req.Msg)
	trends, err := s.services.Dashboard.Trends(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetTrendDataResponse{
		Trends: mapSlice(trends, toProtoTrendPoint),
	}), nil
}

func (s *Server) GetMonthlyComparison(ctx context.Context, req *connect.Request[pb.GetMonthlyComparisonRequest]) (*connect.Response[pb.GetMonthlyComparisonResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := buildMonthlyComparisonParams(userID, req.Msg.MonthsBack)
	comparison, err := s.services.Dashboard.MonthlyComparison(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetMonthlyComparisonResponse{
		Comparisons: mapSlice(comparison, toProtoMonthlyComparison),
	}), nil
}

func (s *Server) GetTopCategories(ctx context.Context, req *connect.Request[pb.GetTopCategoriesRequest]) (*connect.Response[pb.GetTopCategoriesResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := buildTopCategoriesParams(userID, req.Msg)
	categories, err := s.services.Dashboard.TopCategories(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetTopCategoriesResponse{
		Categories: mapSlice(categories, toProtoTopCategory),
	}), nil
}

func (s *Server) GetTopMerchants(ctx context.Context, req *connect.Request[pb.GetTopMerchantsRequest]) (*connect.Response[pb.GetTopMerchantsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := buildTopMerchantsParams(userID, req.Msg)
	merchants, err := s.services.Dashboard.TopMerchants(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetTopMerchantsResponse{
		Merchants: mapSlice(merchants, toProtoTopMerchant),
	}), nil
}

func (s *Server) GetAccountSummary(ctx context.Context, req *connect.Request[pb.GetAccountSummaryRequest]) (*connect.Response[pb.GetAccountSummaryResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	var startDate, endDate *string
	if req.Msg.StartDate != nil {
		start := dateToTime(req.Msg.StartDate).Format("2006-01-02")
		startDate = &start
	}
	if req.Msg.EndDate != nil {
		end := dateToTime(req.Msg.EndDate).Format("2006-01-02")
		endDate = &end
	}

	accountSummary, err := s.services.Dashboard.GetAccountSummary(ctx, userID, req.Msg.GetAccountId(), startDate, endDate)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetAccountSummaryResponse{
		Summary: toProtoDashboardSummaryFromAccount(accountSummary.Summary),
		Trends:  mapSlice(accountSummary.Trends, toProtoTrendPointFromAccount),
	}), nil
}

func (s *Server) GetSpendingTrends(ctx context.Context, req *connect.Request[pb.GetSpendingTrendsRequest]) (*connect.Response[pb.GetSpendingTrendsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	startDate := dateToTime(req.Msg.StartDate).Format("2006-01-02")
	endDate := dateToTime(req.Msg.EndDate).Format("2006-01-02")

	trends, err := s.services.Dashboard.GetSpendingTrends(ctx, userID, startDate, endDate, req.Msg.CategoryId, req.Msg.AccountId)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetSpendingTrendsResponse{
		Trends: mapSlice(trends, toProtoTrendPoint),
	}), nil
}

func (s *Server) GetAccountBalances(ctx context.Context, req *connect.Request[pb.GetAccountBalancesRequest]) (*connect.Response[pb.GetAccountBalancesResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	// get all accounts for the user
	accounts, err := s.services.Accounts.List(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetAccountBalancesResponse{
		Balances: mapSlice(accounts, toProtoAccountBalance),
	}), nil
}

func (s *Server) GetNetBalance(ctx context.Context, req *connect.Request[pb.GetNetBalanceRequest]) (*connect.Response[pb.GetNetBalanceResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	netBalance, err := s.services.Dashboard.NetBalance(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetNetBalanceResponse{
		NetBalance: netBalance,
	}), nil
}

func (s *Server) GetTotalBalance(ctx context.Context, req *connect.Request[pb.GetTotalBalanceRequest]) (*connect.Response[pb.GetTotalBalanceResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	totalBalance, err := s.services.Dashboard.Balance(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetTotalBalanceResponse{
		TotalBalance: totalBalance,
	}), nil
}

func (s *Server) GetTotalDebt(ctx context.Context, req *connect.Request[pb.GetTotalDebtRequest]) (*connect.Response[pb.GetTotalDebtResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	totalDebt, err := s.services.Dashboard.Debt(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetTotalDebtResponse{
		TotalDebt: totalDebt,
	}), nil
}

func (s *Server) GetCategorySpendingComparison(ctx context.Context, req *connect.Request[pb.GetCategorySpendingComparisonRequest]) (*connect.Response[pb.GetCategorySpendingComparisonResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Map proto period type to service period type
	periodType, err := mapPeriodType(req.Msg.PeriodType)
	if err != nil {
		return nil, handleError(err)
	}

	// Convert custom dates if provided (dateToTime from mappers.go returns *time.Time)
	customStart := dateToTime(req.Msg.CustomStartDate)
	customEnd := dateToTime(req.Msg.CustomEndDate)

	// Get spending data from service layer
	result, err := s.services.Dashboard.GetCategorySpendingComparison(ctx, service.CategorySpendingParams{
		UserID:      userID,
		PeriodType:  periodType,
		CustomStart: customStart,
		CustomEnd:   customEnd,
		Timezone:    req.Msg.Timezone,
	})
	if err != nil {
		return nil, handleError(err)
	}

	// Build a map to merge current and previous period data
	type MergedSpending struct {
		CategoryID    *int64
		Slug          *string
		Color         *string
		CurrentCents  int64
		CurrentCount  int64
		PreviousCents int64
		PreviousCount int64
	}

	merged := make(map[string]*MergedSpending)

	// Add current period data
	for i := range result.Current {
		row := &result.Current[i]
		key := categoryKeyToString(row.CategoryID)
		merged[key] = &MergedSpending{
			CategoryID:    row.CategoryID,
			Slug:          row.CategorySlug,
			Color:         row.CategoryColor,
			CurrentCents:  row.TotalCents,
			CurrentCount:  row.TransactionCount,
			PreviousCents: 0,
			PreviousCount: 0,
		}
	}

	// Merge previous period data
	for i := range result.Previous {
		row := &result.Previous[i]
		key := categoryKeyToString(row.CategoryID)
		if existing, ok := merged[key]; ok {
			existing.PreviousCents = row.TotalCents
			existing.PreviousCount = row.TransactionCount
		} else {
			// Category exists in previous but not current
			merged[key] = &MergedSpending{
				CategoryID:    row.CategoryID,
				Slug:          row.CategorySlug,
				Color:         row.CategoryColor,
				CurrentCents:  0,
				CurrentCount:  0,
				PreviousCents: row.TotalCents,
				PreviousCount: row.TransactionCount,
			}
		}
	}

	// Separate categorized from uncategorized and build response
	var categories []*pb.CategorySpendingItem
	var uncategorized *pb.CategorySpendingComparison
	var totalCurrentCents int64
	var totalPreviousCents int64

	for _, spending := range merged {
		currentSpending := &pb.PeriodSpending{
			Amount:           centsToMoney(spending.CurrentCents, "CAD"),
			TransactionCount: spending.CurrentCount,
		}
		previousSpending := &pb.PeriodSpending{
			Amount:           centsToMoney(spending.PreviousCents, "CAD"),
			TransactionCount: spending.PreviousCount,
		}

		comparison := &pb.CategorySpendingComparison{
			CategoryId:     spending.CategoryID,
			CurrentPeriod:  currentSpending,
			PreviousPeriod: previousSpending,
		}

		// Track totals
		totalCurrentCents += spending.CurrentCents
		totalPreviousCents += spending.PreviousCents

		if spending.CategoryID == nil {
			// Uncategorized transactions
			uncategorized = comparison
		} else {
			// Categorized transactions
			item := &pb.CategorySpendingItem{
				Category: &pb.Category{
					Id:    *spending.CategoryID,
					Slug:  *spending.Slug,
					Color: *spending.Color,
				},
				Spending: comparison,
			}
			categories = append(categories, item)
		}
	}

	// Sort categories by current period amount descending
	sortCategoriesByCurrentSpending(categories)

	// Build period info from service result
	currentPeriod := &pb.PeriodInfo{
		StartDate: stringToDate(result.CurrentPeriod.StartDate),
		EndDate:   stringToDate(result.CurrentPeriod.EndDate),
		Label:     result.CurrentPeriod.Label,
	}
	previousPeriod := &pb.PeriodInfo{
		StartDate: stringToDate(result.PreviousPeriod.StartDate),
		EndDate:   stringToDate(result.PreviousPeriod.EndDate),
		Label:     result.PreviousPeriod.Label,
	}

	// Build totals
	totals := &pb.CategorySpendingTotals{
		CurrentPeriodTotal:  centsToMoney(totalCurrentCents, "CAD"),
		PreviousPeriodTotal: centsToMoney(totalPreviousCents, "CAD"),
	}

	return connect.NewResponse(&pb.GetCategorySpendingComparisonResponse{
		CurrentPeriod:  currentPeriod,
		PreviousPeriod: previousPeriod,
		Categories:     categories,
		Uncategorized:  uncategorized,
		Totals:         totals,
	}), nil
}

// categoryKeyToString converts a category ID to a unique string key for the map
func categoryKeyToString(id *int64) string {
	if id == nil {
		return "uncategorized"
	}
	return fmt.Sprintf("cat_%d", *id)
}

// sortCategoriesByCurrentSpending sorts categories by current period spending descending
func sortCategoriesByCurrentSpending(categories []*pb.CategorySpendingItem) {
	sort.Slice(categories, func(i, j int) bool {
		iCents := categories[i].Spending.CurrentPeriod.Amount.Units*100 +
			int64(categories[i].Spending.CurrentPeriod.Amount.Nanos/10000000)
		jCents := categories[j].Spending.CurrentPeriod.Amount.Units*100 +
			int64(categories[j].Spending.CurrentPeriod.Amount.Nanos/10000000)
		return iCents > jCents
	})
}

// mapPeriodType converts proto PeriodType to service PeriodType
func mapPeriodType(pt pb.PeriodType) (service.PeriodType, error) {
	switch pt {
	case pb.PeriodType_PERIOD_TYPE_7_DAYS:
		return service.Period7Days, nil
	case pb.PeriodType_PERIOD_TYPE_30_DAYS:
		return service.Period30Days, nil
	case pb.PeriodType_PERIOD_TYPE_90_DAYS:
		return service.Period90Days, nil
	case pb.PeriodType_PERIOD_TYPE_CUSTOM:
		return service.PeriodCustom, nil
	default:
		return 0, fmt.Errorf("invalid period type: %v", pt)
	}
}

// stringToDate converts a date string (YYYY-MM-DD) to google.type.Date
func stringToDate(s string) *date.Date {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &date.Date{
		Year:  int32(t.Year()),
		Month: int32(t.Month()),
		Day:   int32(t.Day()),
	}
}
