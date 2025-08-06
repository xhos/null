package grpc

import (
	sqlc "ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Helper function to convert google.type.Date to time.Time
func dateToTime(d *date.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := time.Date(int(d.Year), time.Month(d.Month), int(d.Day), 0, 0, 0, 0, time.UTC)
	return &t
}

// Helper function to convert time.Time to google.type.Date
func timeToDate(t time.Time) *date.Date {
	if t.IsZero() {
		return nil
	}
	return &date.Date{
		Year:  int32(t.Year()),
		Month: int32(t.Month()),
		Day:   int32(t.Day()),
	}
}

// Helper to safely convert interface{} to decimal.Decimal
func interfaceToDecimal(v interface{}) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	switch val := v.(type) {
	case decimal.Decimal:
		return val
	case float64:
		return decimal.NewFromFloat(val)
	case int64:
		return decimal.NewFromInt(val)
	default:
		return decimal.Zero
	}
}

// ==================== DASHBOARD SERVICE ====================

func (s *Server) GetDashboardSummary(ctx context.Context, req *pb.GetDashboardSummaryRequest) (*pb.GetDashboardSummaryResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	var start, end *time.Time
	if req.StartDate != nil {
		start = dateToTime(req.StartDate)
	}
	if req.EndDate != nil {
		end = dateToTime(req.EndDate)
	}

	params := sqlc.GetDashboardSummaryForUserParams{
		UserID: userID,
		Start:  start,
		End:    end,
	}

	summary, err := s.services.Dashboard.SummaryForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetDashboardSummaryResponse{
		Summary: &pb.DashboardSummary{
			TotalAccounts:             summary.TotalAccounts,
			TotalTransactions:         summary.TotalTransactions,
			TotalIncome:               decimalToMoney(interfaceToDecimal(summary.TotalIncome), "CAD"),
			TotalExpenses:             decimalToMoney(interfaceToDecimal(summary.TotalExpenses), "CAD"),
			UncategorizedTransactions: summary.UncategorizedTransactions,
		},
	}, nil
}

func (s *Server) GetTrendData(ctx context.Context, req *pb.GetTrendDataRequest) (*pb.GetTrendDataResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	var start, end *time.Time
	if req.StartDate != nil {
		start = dateToTime(req.StartDate)
	}
	if req.EndDate != nil {
		end = dateToTime(req.EndDate)
	}

	params := sqlc.GetDashboardTrendsForUserParams{
		UserID: userID,
		Start:  start,
		End:    end,
	}

	trends, err := s.services.Dashboard.TrendsForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	pbTrends := make([]*pb.TrendPoint, len(trends))
	for i, trend := range trends {
		// Parse the date string to time.Time for conversion
		trendDate, _ := time.Parse("2006-01-02", trend.Date)
		pbTrends[i] = &pb.TrendPoint{
			Date:     timeToDate(trendDate),
			Income:   decimalToMoney(decimal.NewFromInt(trend.Income), "CAD"),
			Expenses: decimalToMoney(decimal.NewFromInt(trend.Expenses), "CAD"),
		}
	}

	return &pb.GetTrendDataResponse{
		Trends: pbTrends,
	}, nil
}

func (s *Server) GetMonthlyComparison(ctx context.Context, req *pb.GetMonthlyComparisonRequest) (*pb.GetMonthlyComparisonResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	// Calculate date range based on months_back
	end := time.Now()
	start := end.AddDate(0, -int(req.MonthsBack), 0)

	params := sqlc.GetMonthlyComparisonForUserParams{
		UserID: userID,
		Start:  &start,
		End:    &end,
	}

	comparison, err := s.services.Dashboard.MonthlyComparisonForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	pbComparison := make([]*pb.MonthlyComparison, len(comparison))
	for i, comp := range comparison {
		pbComparison[i] = &pb.MonthlyComparison{
			Month:    comp.Month,
			Income:   decimalToMoney(decimal.NewFromInt(comp.Income), "CAD"),
			Expenses: decimalToMoney(decimal.NewFromInt(comp.Expenses), "CAD"),
			Net:      decimalToMoney(decimal.NewFromInt(comp.Net), "CAD"),
		}
	}

	return &pb.GetMonthlyComparisonResponse{
		Comparisons: pbComparison,
	}, nil
}

func (s *Server) GetTopCategories(ctx context.Context, req *pb.GetTopCategoriesRequest) (*pb.GetTopCategoriesResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	var start, end *time.Time
	if req.StartDate != nil {
		start = dateToTime(req.StartDate)
	}
	if req.EndDate != nil {
		end = dateToTime(req.EndDate)
	}

	params := sqlc.GetTopCategoriesForUserParams{
		UserID: userID,
		Start:  start,
		End:    end,
		Limit:  req.Limit,
	}

	categories, err := s.services.Dashboard.TopCategoriesForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	pbCategories := make([]*pb.TopCategory, len(categories))
	for i, cat := range categories {
		pbCategories[i] = &pb.TopCategory{
			Slug:             cat.Slug,
			Label:            cat.Label,
			Color:            cat.Color,
			TransactionCount: cat.TransactionCount,
			TotalAmount:      decimalToMoney(decimal.NewFromInt(cat.TotalAmount), "CAD"),
		}
	}

	return &pb.GetTopCategoriesResponse{
		Categories: pbCategories,
	}, nil
}

func (s *Server) GetTopMerchants(ctx context.Context, req *pb.GetTopMerchantsRequest) (*pb.GetTopMerchantsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	var start, end *time.Time
	if req.StartDate != nil {
		start = dateToTime(req.StartDate)
	}
	if req.EndDate != nil {
		end = dateToTime(req.EndDate)
	}

	params := sqlc.GetTopMerchantsForUserParams{
		UserID: userID,
		Start:  start,
		End:    end,
		Limit:  req.Limit,
	}

	merchants, err := s.services.Dashboard.TopMerchantsForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	pbMerchants := make([]*pb.TopMerchant, len(merchants))
	for i, merchant := range merchants {
		merchantName := ""
		if merchant.Merchant != nil {
			merchantName = *merchant.Merchant
		}

		pbMerchants[i] = &pb.TopMerchant{
			Merchant:         merchantName,
			TransactionCount: merchant.TransactionCount,
			TotalAmount:      decimalToMoney(decimal.NewFromInt(merchant.TotalAmount), "CAD"),
			AvgAmount:        decimalToMoney(decimal.NewFromFloat(merchant.AvgAmount), "CAD"),
		}
	}

	return &pb.GetTopMerchantsResponse{
		Merchants: pbMerchants,
	}, nil
}

func (s *Server) GetAccountSummary(ctx context.Context, req *pb.GetAccountSummaryRequest) (*pb.GetAccountSummaryResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account_id must be positive")
	}

	var startDate, endDate *string
	if req.StartDate != nil {
		start := dateToTime(req.StartDate).Format("2006-01-02")
		startDate = &start
	}
	if req.EndDate != nil {
		end := dateToTime(req.EndDate).Format("2006-01-02")
		endDate = &end
	}

	accountSummary, err := s.services.Dashboard.GetAccountSummary(ctx, userID, req.GetAccountId(), startDate, endDate)
	if err != nil {
		return nil, handleError(err)
	}

	// Extract summary data
	summaryRow, ok := accountSummary.Summary.(*sqlc.GetDashboardSummaryForAccountRow)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get account summary data")
	}

	// Extract trends data
	trendsRows, ok := accountSummary.Trends.([]sqlc.GetDashboardTrendsForAccountRow)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get account trends data")
	}

	pbTrends := make([]*pb.TrendPoint, len(trendsRows))
	for i, trend := range trendsRows {
		// Parse the date string to time.Time for conversion
		trendDate, _ := time.Parse("2006-01-02", trend.Date)
		pbTrends[i] = &pb.TrendPoint{
			Date:     timeToDate(trendDate),
			Income:   decimalToMoney(decimal.NewFromInt(trend.Income), "CAD"),
			Expenses: decimalToMoney(decimal.NewFromInt(trend.Expenses), "CAD"),
		}
	}

	return &pb.GetAccountSummaryResponse{
		Summary: &pb.DashboardSummary{
			TotalAccounts:             summaryRow.TotalAccounts,
			TotalTransactions:         summaryRow.TotalTransactions,
			TotalIncome:               decimalToMoney(interfaceToDecimal(summaryRow.TotalIncome), "CAD"),
			TotalExpenses:             decimalToMoney(interfaceToDecimal(summaryRow.TotalExpenses), "CAD"),
			UncategorizedTransactions: summaryRow.UncategorizedTransactions,
		},
		Trends: pbTrends,
	}, nil
}

func (s *Server) GetSpendingTrends(ctx context.Context, req *pb.GetSpendingTrendsRequest) (*pb.GetSpendingTrendsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.StartDate == nil || req.EndDate == nil {
		return nil, status.Error(codes.InvalidArgument, "start_date and end_date are required")
	}

	startDate := dateToTime(req.StartDate).Format("2006-01-02")
	endDate := dateToTime(req.EndDate).Format("2006-01-02")

	trends, err := s.services.Dashboard.GetSpendingTrends(ctx, userID, startDate, endDate, req.CategoryId, req.AccountId)
	if err != nil {
		return nil, handleError(err)
	}

	pbTrends := make([]*pb.TrendPoint, len(trends))
	for i, trend := range trends {
		// Parse the date string to time.Time for conversion
		trendDate, _ := time.Parse("2006-01-02", trend.Date)
		pbTrends[i] = &pb.TrendPoint{
			Date:     timeToDate(trendDate),
			Income:   decimalToMoney(decimal.NewFromInt(trend.Income), "CAD"),
			Expenses: decimalToMoney(decimal.NewFromInt(trend.Expenses), "CAD"),
		}
	}

	return &pb.GetSpendingTrendsResponse{
		Trends: pbTrends,
	}, nil
}
