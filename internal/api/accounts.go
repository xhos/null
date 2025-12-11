package api

import (
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"connectrpc.com/connect"
)

func (s *Server) ListAccounts(ctx context.Context, req *connect.Request[pb.ListAccountsRequest]) (*connect.Response[pb.ListAccountsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := s.services.Accounts.List(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ListAccountsResponse{
		Accounts: mapSlice(accounts, toProtoAccount),
	}), nil
}

func (s *Server) GetAccount(ctx context.Context, req *connect.Request[pb.GetAccountRequest]) (*connect.Response[pb.GetAccountResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	account, err := s.services.Accounts.Get(ctx, userID, req.Msg.GetId())
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetAccountResponse{
		Account: toProtoAccount(account),
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
		Account: toProtoAccount(account),
	}), nil
}

func (s *Server) UpdateAccount(ctx context.Context, req *connect.Request[pb.UpdateAccountRequest]) (*connect.Response[pb.UpdateAccountResponse], error) {
	params := buildUpdateAccountParams(req.Msg)
	account, err := s.services.Accounts.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateAccountResponse{
		Account: toProtoAccount(account),
	}), nil
}

func (s *Server) DeleteAccount(ctx context.Context, req *connect.Request[pb.DeleteAccountRequest]) (*connect.Response[pb.DeleteAccountResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.DeleteAccountParams{
		UserID: userID,
		ID:     req.Msg.GetId(),
	}

	affectedRows, err := s.services.Accounts.Delete(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteAccountResponse{
		AffectedRows: affectedRows,
	}), nil
}

func (s *Server) SetAccountAnchor(ctx context.Context, req *connect.Request[pb.SetAccountAnchorRequest]) (*connect.Response[pb.SetAccountAnchorResponse], error) {
	balance := req.Msg.GetBalance()
	cents := moneyToCents(balance)
	currency := balance.GetCurrencyCode()

	params := sqlc.SetAccountAnchorParams{
		ID:                 req.Msg.GetId(),
		AnchorBalanceCents: cents,
		AnchorCurrency:     currency,
	}

	err := s.services.Accounts.SetAnchor(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.SetAccountAnchorResponse{
		AffectedRows: 1,
	}), nil
}

func (s *Server) GetAccountsCount(ctx context.Context, req *connect.Request[pb.GetAccountsCountRequest]) (*connect.Response[pb.GetAccountsCountResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	count, err := s.services.Accounts.GetAccountCount(ctx, userID)
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
