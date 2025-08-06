package grpc

import (
	sqlc "ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ==================== ACCOUNT SERVICE ====================

func (s *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	accounts, err := s.services.Accounts.ListForUser(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	pbAccounts := make([]*pb.Account, len(accounts))
	for i, account := range accounts {
		pbAccounts[i] = toProtoAccount(&account)
	}

	return &pb.ListAccountsResponse{
		Accounts: pbAccounts,
	}, nil
}

func (s *Server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account id must be positive")
	}

	account, err := s.services.Accounts.GetForUser(ctx, sqlc.GetAccountForUserParams{
		UserID: userID,
		ID:     req.GetId(),
	})
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetAccountResponse{
		Account: toProtoAccountFromGetRow(account),
	}, nil
}

func (s *Server) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.GetBank() == "" {
		return nil, status.Error(codes.InvalidArgument, "bank is required")
	}
	if req.GetAnchorBalance() == nil {
		return nil, status.Error(codes.InvalidArgument, "anchor_balance is required")
	}

	params, err := createAccountParamsFromProto(req)
	if err != nil {
		return nil, err
	}

	account, err := s.services.Accounts.Create(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.CreateAccountResponse{
		Account: toProtoAccountFromModel(account),
	}, nil
}

func (s *Server) UpdateAccount(ctx context.Context, req *pb.UpdateAccountRequest) (*pb.UpdateAccountResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account id must be positive")
	}

	// build update params based on field mask
	params := sqlc.UpdateAccountParams{
		ID: req.GetId(),
	}

	// apply updates based on what's provided
	if req.Name != nil {
		params.Name = req.Name
	}
	if req.Bank != nil {
		params.Bank = req.Bank
	}
	if req.AccountType != nil {
		accountType := int16(*req.AccountType)
		params.AccountType = &accountType
	}
	if req.Alias != nil {
		params.Alias = req.Alias
	}
	if req.AnchorDate != nil {
		// Convert timestamppb.Timestamp to date.Date
		t := req.AnchorDate.AsTime()
		params.AnchorDate = &date.Date{
			Year:  int32(t.Year()),
			Month: int32(t.Month()),
			Day:   int32(t.Day()),
		}
	}
	if req.AnchorBalance != nil {
		balance := moneyToDecimal(req.AnchorBalance)
		currency := req.AnchorBalance.CurrencyCode
		params.AnchorBalance = balance
		params.AnchorCurrency = &currency
	}

	account, err := s.services.Accounts.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.UpdateAccountResponse{
		Account: toProtoAccountFromModel(account),
	}, nil
}

func (s *Server) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account id must be positive")
	}

	err = s.services.Accounts.DeleteForUser(ctx, sqlc.DeleteAccountForUserParams{
		UserID: userID,
		ID:     req.GetId(),
	})
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.DeleteAccountResponse{
		AffectedRows: 1, // Always 1 for single delete
	}, nil
}

func (s *Server) SetAccountAnchor(ctx context.Context, req *pb.SetAccountAnchorRequest) (*pb.SetAccountAnchorResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account id must be positive")
	}
	if req.GetBalance() == nil {
		return nil, status.Error(codes.InvalidArgument, "balance is required")
	}

	balance := moneyToDecimal(req.GetBalance())
	currency := req.GetBalance().CurrencyCode

	// for anchor setting, we need to update the account
	// note: this method doesn't require user ownership check in the original proto
	_, err := s.services.Accounts.Update(ctx, sqlc.UpdateAccountParams{
		ID:             req.GetId(),
		AnchorBalance:  balance,
		AnchorCurrency: &currency,
	})
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.SetAccountAnchorResponse{
		AffectedRows: 1,
	}, nil
}

func (s *Server) GetAccountBalance(ctx context.Context, req *pb.GetAccountBalanceRequest) (*pb.GetAccountBalanceResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account id must be positive")
	}

	balance, err := s.services.Accounts.GetBalance(ctx, req.GetId())
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetAccountBalanceResponse{
		Balance: balance,
	}, nil
}

func (s *Server) GetAccountsCount(ctx context.Context, req *pb.GetAccountsCountRequest) (*pb.GetAccountsCountResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	count, err := s.services.Accounts.GetUserAccountsCount(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetAccountsCountResponse{
		Count: count,
	}, nil
}

func (s *Server) SyncAccountBalances(ctx context.Context, req *pb.SyncAccountBalancesRequest) (*pb.SyncAccountBalancesResponse, error) {
	// this is a placeholder - the actual implementation would depend on
	// how balance syncing is supposed to work in the business logic
	s.log.Info("SyncAccountBalances called", "account_id", req.GetAccountId())

	// for now, just return success without doing anything
	return &pb.SyncAccountBalancesResponse{}, nil
}
