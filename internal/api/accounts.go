package api

import (
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/type/money"
)

func (s *Server) ListAccounts(ctx context.Context, req *connect.Request[pb.ListAccountsRequest]) (*connect.Response[pb.ListAccountsResponse], error) {
	userID, err := getUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := s.services.Accounts.ListForUser(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ListAccountsResponse{
		Accounts: mapSlice(accounts, toProtoAccount),
	}), nil
}

func (s *Server) GetAccount(ctx context.Context, req *connect.Request[pb.GetAccountRequest]) (*connect.Response[pb.GetAccountResponse], error) {
	userID, err := getUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	account, err := s.services.Accounts.GetForUser(ctx, userID, req.Msg.GetId())
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetAccountResponse{
		Account: toProtoAccountFromGetRow(account),
	}), nil
}

func (s *Server) CreateAccount(ctx context.Context, req *connect.Request[pb.CreateAccountRequest]) (*connect.Response[pb.CreateAccountResponse], error) {
	params, err := createAccountParamsFromProto(req.Msg)
	if err != nil {
		return nil, err
	}

	account, err := s.services.Accounts.Create(ctx, params, s.services.Users)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.CreateAccountResponse{
		Account: toProtoAccountFromModel(account),
	}), nil
}

func (s *Server) UpdateAccount(ctx context.Context, req *connect.Request[pb.UpdateAccountRequest]) (*connect.Response[pb.UpdateAccountResponse], error) {
	params := buildUpdateAccountParams(req.Msg)
	account, err := s.services.Accounts.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateAccountResponse{
		Account: toProtoAccountFromModel(account),
	}), nil
}

func (s *Server) DeleteAccount(ctx context.Context, req *connect.Request[pb.DeleteAccountRequest]) (*connect.Response[pb.DeleteAccountResponse], error) {
	userID, err := getUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.DeleteAccountForUserParams{
		UserID: userID,
		ID:     req.Msg.GetId(),
	}

	affectedRows, err := s.services.Accounts.DeleteForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteAccountResponse{
		AffectedRows: affectedRows,
	}), nil
}

func (s *Server) SetAccountAnchor(ctx context.Context, req *connect.Request[pb.SetAccountAnchorRequest]) (*connect.Response[pb.SetAccountAnchorResponse], error) {
	balance := moneyToDecimal(req.Msg.GetBalance())
	currency := req.Msg.GetBalance().CurrencyCode

	params := sqlc.UpdateAccountParams{
		ID:             req.Msg.GetId(),
		AnchorBalance:  balance,
		AnchorCurrency: &currency,
	}

	_, err := s.services.Accounts.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.SetAccountAnchorResponse{
		AffectedRows: 1,
	}), nil
}

func (s *Server) GetAccountBalance(ctx context.Context, req *connect.Request[pb.GetAccountBalanceRequest]) (*connect.Response[pb.GetAccountBalanceResponse], error) {
	balance, err := s.services.Accounts.GetBalance(ctx, req.Msg.GetId())
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetAccountBalanceResponse{
		Balance: balance,
	}), nil
}

func (s *Server) GetAccountsCount(ctx context.Context, req *connect.Request[pb.GetAccountsCountRequest]) (*connect.Response[pb.GetAccountsCountResponse], error) {
	userID, err := getUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	count, err := s.services.Accounts.GetUserAccountsCount(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetAccountsCountResponse{
		Count: count,
	}), nil
}

func (s *Server) SyncAccountBalances(ctx context.Context, req *connect.Request[pb.SyncAccountBalancesRequest]) (*connect.Response[pb.SyncAccountBalancesResponse], error) {
	// placeholder - delegate to service layer when implemented
	s.log.Info("SyncAccountBalances called", "account_id", req.Msg.GetAccountId())

	return connect.NewResponse(&pb.SyncAccountBalancesResponse{}), nil
}

func (s *Server) CheckUserAccountAccess(ctx context.Context, req *connect.Request[pb.CheckUserAccountAccessRequest]) (*connect.Response[pb.CheckUserAccountAccessResponse], error) {
	userID, err := getUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// check if user has access to this account (either owner or collaborator)
	_, err = s.services.Accounts.GetForUser(ctx, userID, req.Msg.GetAccountId())
	hasAccess := err == nil

	return connect.NewResponse(&pb.CheckUserAccountAccessResponse{
		HasAccess: hasAccess,
	}), nil
}

func (s *Server) GetAnchorBalance(ctx context.Context, req *connect.Request[pb.GetAnchorBalanceRequest]) (*connect.Response[pb.GetAnchorBalanceResponse], error) {
	// placeholder implementation - would need to fetch anchor balance from database
	return connect.NewResponse(&pb.GetAnchorBalanceResponse{
		AnchorBalance: &money.Money{CurrencyCode: "CAD", Units: 0, Nanos: 0},
		Currency:      "CAD",
	}), nil
}
