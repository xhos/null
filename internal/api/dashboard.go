package api

import (
	pb "ariand/internal/gen/arian/v1"
	"context"

	"connectrpc.com/connect"
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
	accounts, err := s.services.Accounts.ListForUser(ctx, userID)
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
		NetBalance: decimalToMoney(netBalance, "CAD"),
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
		TotalBalance: decimalToMoney(totalBalance, "CAD"),
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
		TotalDebt: decimalToMoney(totalDebt, "CAD"),
	}), nil
}
