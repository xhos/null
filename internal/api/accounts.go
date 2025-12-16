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
		Accounts: accounts,
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
		Account: account,
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
		Account: account,
	}), nil
}

func (s *Server) UpdateAccount(ctx context.Context, req *connect.Request[pb.UpdateAccountRequest]) (*connect.Response[pb.UpdateAccountResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := buildUpdateAccountParams(req.Msg)
	params.UserID = userID

	err = s.services.Accounts.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateAccountResponse{}), nil
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
