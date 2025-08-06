package grpc

import (
	sqlc "ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ==================== TRANSACTION SERVICE ====================

func (s *Server) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	params := sqlc.ListTransactionsForUserParams{
		UserID: userID,
		Limit:  req.Limit,
	}

	// Handle cursor pagination
	if cursor := req.GetCursor(); cursor != nil {
		if cursor.Date != nil {
			cursorTime := fromProtoTimestamp(cursor.Date)
			params.CursorDate = &cursorTime
		}
		if cursor.Id != nil {
			params.CursorID = cursor.Id
		}
	}

	// Handle filters
	if req.StartDate != nil {
		startTime := fromProtoTimestamp(req.StartDate)
		params.Start = &startTime
	}
	if req.EndDate != nil {
		endTime := fromProtoTimestamp(req.EndDate)
		params.End = &endTime
	}
	if req.AmountMin != nil {
		params.AmountMin = moneyToDecimal(req.AmountMin)
	}
	if req.AmountMax != nil {
		params.AmountMax = moneyToDecimal(req.AmountMax)
	}
	if req.Direction != nil {
		direction := int16(*req.Direction)
		params.Direction = &direction
	}
	if len(req.AccountIds) > 0 {
		params.AccountIds = req.AccountIds
	}
	if len(req.Categories) > 0 {
		params.Categories = req.Categories
	}
	if req.MerchantQuery != nil {
		params.MerchantQ = req.MerchantQuery
	}
	if req.DescriptionQuery != nil {
		params.DescQ = req.DescriptionQuery
	}
	if req.Currency != nil {
		params.Currency = req.Currency
	}

	transactions, err := s.services.Transactions.ListForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	pbTransactions := make([]*pb.Transaction, len(transactions))
	for i, tx := range transactions {
		pbTransactions[i] = toProtoTransactionFromListRow(&tx)
	}

	// Create next cursor from last transaction
	var nextCursor *pb.Cursor
	if len(transactions) > 0 && req.Limit != nil && len(transactions) == int(*req.Limit) {
		lastTx := transactions[len(transactions)-1]
		nextCursor = &pb.Cursor{
			Date: toProtoTimestamp(&lastTx.TxDate),
			Id:   &lastTx.ID,
		}
	}

	return &pb.ListTransactionsResponse{
		Transactions: pbTransactions,
		TotalCount:   int64(len(transactions)), // Note: This would need a separate count query for true total
		NextCursor:   nextCursor,
	}, nil
}

func (s *Server) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "transaction id must be positive")
	}

	params := sqlc.GetTransactionForUserParams{
		UserID: userID,
		ID:     req.GetId(),
	}

	transaction, err := s.services.Transactions.GetForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetTransactionResponse{
		Transaction: toProtoTransactionFromGetRow(transaction),
	}, nil
}

func (s *Server) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.CreateTransactionResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account_id must be positive")
	}

	if req.TxAmount == nil {
		return nil, status.Error(codes.InvalidArgument, "tx_amount is required")
	}

	if req.TxDate == nil {
		return nil, status.Error(codes.InvalidArgument, "tx_date is required")
	}

	params := sqlc.CreateTransactionForUserParams{
		UserID:      userID,
		AccountID:   req.GetAccountId(),
		TxDate:      fromProtoTimestamp(req.TxDate),
		TxAmount:    *moneyToDecimal(req.TxAmount),
		TxDirection: int16(req.Direction),
		TxDesc:      req.Description,
		Merchant:    req.Merchant,
		UserNotes:   req.UserNotes,
	}

	if req.CategoryId != nil {
		params.CategoryID = req.CategoryId
	}

	if req.ForeignAmount != nil {
		params.ForeignAmount = moneyToDecimal(req.ForeignAmount)
		if req.ExchangeRate != nil {
			exchangeRate := decimal.NewFromFloat(*req.ExchangeRate)
			params.ExchangeRate = &exchangeRate
		}
	}

	transactionID, err := s.services.Transactions.CreateForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// Get the created transaction
	getParams := sqlc.GetTransactionForUserParams{
		UserID: userID,
		ID:     transactionID,
	}

	transaction, err := s.services.Transactions.GetForUser(ctx, getParams)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.CreateTransactionResponse{
		Transaction: toProtoTransactionFromGetRow(transaction),
	}, nil
}

func (s *Server) UpdateTransaction(ctx context.Context, req *pb.UpdateTransactionRequest) (*pb.UpdateTransactionResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "transaction id must be positive")
	}

	params := sqlc.UpdateTransactionParams{
		ID:     req.GetId(),
		UserID: userID,
	}

	// Apply field mask updates
	if req.TxDate != nil {
		txTime := fromProtoTimestamp(req.TxDate)
		params.TxDate = &txTime
	}
	if req.TxAmount != nil {
		amount := moneyToDecimal(req.TxAmount)
		params.TxAmount = amount
	}
	if req.Direction != nil {
		direction := int16(*req.Direction)
		params.TxDirection = &direction
	}
	if req.Description != nil {
		params.TxDesc = req.Description
	}
	if req.Merchant != nil {
		params.Merchant = req.Merchant
	}
	if req.UserNotes != nil {
		params.UserNotes = req.UserNotes
	}
	if req.CategoryId != nil {
		params.CategoryID = req.CategoryId
	}
	if req.ForeignAmount != nil {
		params.ForeignAmount = moneyToDecimal(req.ForeignAmount)
	}
	if req.ExchangeRate != nil {
		exchangeRate := decimal.NewFromFloat(*req.ExchangeRate)
		params.ExchangeRate = &exchangeRate
	}

	err = s.services.Transactions.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// Get the updated transaction
	getParams := sqlc.GetTransactionForUserParams{
		UserID: userID,
		ID:     req.GetId(),
	}

	transaction, err := s.services.Transactions.GetForUser(ctx, getParams)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.UpdateTransactionResponse{
		Transaction: toProtoTransactionFromGetRow(transaction),
	}, nil
}

func (s *Server) DeleteTransaction(ctx context.Context, req *pb.DeleteTransactionRequest) (*pb.DeleteTransactionResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "transaction id must be positive")
	}

	params := sqlc.DeleteTransactionForUserParams{
		UserID: userID,
		ID:     req.GetId(),
	}

	affectedRows, err := s.services.Transactions.DeleteForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.DeleteTransactionResponse{
		AffectedRows: affectedRows,
	}, nil
}

func (s *Server) BulkDeleteTransactions(ctx context.Context, req *pb.BulkDeleteTransactionsRequest) (*pb.BulkDeleteTransactionsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if len(req.TransactionIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "transaction_ids cannot be empty")
	}

	params := sqlc.BulkDeleteTransactionsForUserParams{
		UserID:         userID,
		TransactionIds: req.TransactionIds,
	}

	err = s.services.Transactions.BulkDeleteForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.BulkDeleteTransactionsResponse{
		AffectedRows: int64(len(req.TransactionIds)),
	}, nil
}

func (s *Server) CategorizeTransaction(ctx context.Context, req *pb.CategorizeTransactionRequest) (*pb.CategorizeTransactionResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetTransactionId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "transaction_id must be positive")
	}

	if req.GetCategoryId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "category_id must be positive")
	}

	// Manual categorization via bulk categorize
	params := sqlc.BulkCategorizeTransactionsForUserParams{
		UserID:         userID,
		TransactionIds: []int64{req.GetTransactionId()},
		CategoryID:     req.GetCategoryId(),
	}

	err = s.services.Transactions.BulkCategorizeForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// Get the updated transaction
	getParams := sqlc.GetTransactionForUserParams{
		UserID: userID,
		ID:     req.GetTransactionId(),
	}

	transaction, err := s.services.Transactions.GetForUser(ctx, getParams)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.CategorizeTransactionResponse{
		Transaction: toProtoTransactionFromGetRow(transaction),
	}, nil
}

func (s *Server) SearchTransactions(ctx context.Context, req *pb.SearchTransactionsRequest) (*pb.SearchTransactionsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetQuery() == "" {
		return nil, status.Error(codes.InvalidArgument, "query cannot be empty")
	}

	transactions, err := s.services.Transactions.SearchTransactions(ctx, userID, req.GetQuery(), req.AccountId, req.CategoryId, req.Limit, req.Offset)
	if err != nil {
		return nil, handleError(err)
	}

	// Convert to TransactionWithScore (reuse the same data with score = 1.0)
	pbTransactions := make([]*pb.TransactionWithScore, len(transactions))
	for i, tx := range transactions {
		pbTransactions[i] = &pb.TransactionWithScore{
			Transaction: toProtoTransactionFromListRow(&tx),
			// Score field removed from proto, skip it
		}
	}

	return &pb.SearchTransactionsResponse{
		Transactions: pbTransactions,
		TotalCount:   int64(len(transactions)),
	}, nil
}

func (s *Server) GetTransactionsByAccount(ctx context.Context, req *pb.GetTransactionsByAccountRequest) (*pb.GetTransactionsByAccountResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account_id must be positive")
	}

	transactions, err := s.services.Transactions.GetTransactionsByAccount(ctx, userID, req.GetAccountId(), req.Limit, req.Offset)
	if err != nil {
		return nil, handleError(err)
	}

	pbTransactions := make([]*pb.Transaction, len(transactions))
	for i, tx := range transactions {
		pbTransactions[i] = toProtoTransactionFromListRow(&tx)
	}

	// Create next cursor from last transaction
	var nextCursor *pb.Cursor
	if len(transactions) > 0 && req.Limit != nil && len(transactions) == int(*req.Limit) {
		lastTx := transactions[len(transactions)-1]
		nextCursor = &pb.Cursor{
			Date: toProtoTimestamp(&lastTx.TxDate),
			Id:   &lastTx.ID,
		}
	}

	return &pb.GetTransactionsByAccountResponse{
		Transactions: pbTransactions,
		TotalCount:   int64(len(transactions)),
		NextCursor:   nextCursor,
	}, nil
}

func (s *Server) GetUncategorizedTransactions(ctx context.Context, req *pb.GetUncategorizedTransactionsRequest) (*pb.GetUncategorizedTransactionsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	transactions, err := s.services.Transactions.GetUncategorizedTransactions(ctx, userID, req.AccountId, req.Limit, req.Offset)
	if err != nil {
		return nil, handleError(err)
	}

	pbTransactions := make([]*pb.Transaction, len(transactions))
	for i, tx := range transactions {
		pbTransactions[i] = toProtoTransactionFromListRow(&tx)
	}

	// Create next cursor from last transaction
	var nextCursor *pb.Cursor
	if len(transactions) > 0 && req.Limit != nil && len(transactions) == int(*req.Limit) {
		lastTx := transactions[len(transactions)-1]
		nextCursor = &pb.Cursor{
			Date: toProtoTimestamp(&lastTx.TxDate),
			Id:   &lastTx.ID,
		}
	}

	return &pb.GetUncategorizedTransactionsResponse{
		Transactions: pbTransactions,
		TotalCount:   int64(len(transactions)),
		NextCursor:   nextCursor,
	}, nil
}

func (s *Server) BulkCategorizeTransactions(ctx context.Context, req *pb.BulkCategorizeTransactionsRequest) (*pb.BulkCategorizeTransactionsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if len(req.TransactionIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "transaction_ids cannot be empty")
	}

	if req.GetCategoryId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "category_id must be positive")
	}

	params := sqlc.BulkCategorizeTransactionsForUserParams{
		UserID:         userID,
		TransactionIds: req.TransactionIds,
		CategoryID:     req.GetCategoryId(),
	}

	err = s.services.Transactions.BulkCategorizeForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.BulkCategorizeTransactionsResponse{
		AffectedRows: int64(len(req.TransactionIds)),
	}, nil
}

func (s *Server) GetTransactionCountByAccount(ctx context.Context, req *pb.GetTransactionCountByAccountRequest) (*pb.GetTransactionCountByAccountResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	counts, err := s.services.Transactions.GetTransactionCountByAccountForUser(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	pbCounts := make([]*pb.TransactionCountByAccount, len(counts))
	for i, count := range counts {
		pbCounts[i] = &pb.TransactionCountByAccount{
			AccountId:        count.ID,
			TransactionCount: count.TransactionCount,
		}
	}

	return &pb.GetTransactionCountByAccountResponse{
		Counts: pbCounts,
	}, nil
}

func (s *Server) FindCandidateTransactions(ctx context.Context, req *pb.FindCandidateTransactionsRequest) (*pb.FindCandidateTransactionsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetMerchant() == "" {
		return nil, status.Error(codes.InvalidArgument, "merchant cannot be empty")
	}

	if req.PurchaseDate == nil {
		return nil, status.Error(codes.InvalidArgument, "purchase_date is required")
	}

	if req.TotalAmount == nil {
		return nil, status.Error(codes.InvalidArgument, "total_amount is required")
	}

	params := sqlc.FindCandidateTransactionsForUserParams{
		UserID:   userID,
		Merchant: req.GetMerchant(),
		Date:     timestampToDate(req.PurchaseDate),
		Total:    *moneyToDecimal(req.TotalAmount),
	}

	candidates, err := s.services.Transactions.FindCandidateTransactionsForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	pbCandidates := make([]*pb.TransactionWithScore, len(candidates))
	for i, candidate := range candidates {
		pbCandidates[i] = &pb.TransactionWithScore{
			Transaction: toProtoTransactionFromFindRow(&candidate),
			// Score field removed from proto, skip it
		}
	}

	return &pb.FindCandidateTransactionsResponse{
		Candidates: pbCandidates,
	}, nil
}

func (s *Server) IdentifyMerchant(ctx context.Context, req *pb.IdentifyMerchantRequest) (*pb.IdentifyMerchantResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetTransactionId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "transaction_id must be positive")
	}

	err = s.services.Transactions.IdentifyMerchantForTransaction(ctx, userID, req.GetTransactionId())
	if err != nil {
		return nil, handleError(err)
	}

	// Get the updated transaction to return the identified merchant
	getParams := sqlc.GetTransactionForUserParams{
		UserID: userID,
		ID:     req.GetTransactionId(),
	}

	transaction, err := s.services.Transactions.GetForUser(ctx, getParams)
	if err != nil {
		return nil, handleError(err)
	}

	merchant := ""
	if transaction.Merchant != nil {
		merchant = *transaction.Merchant
	}

	return &pb.IdentifyMerchantResponse{
		Merchant: merchant,
	}, nil
}

func (s *Server) SetTransactionReceipt(ctx context.Context, req *pb.SetTransactionReceiptRequest) (*pb.SetTransactionReceiptResponse, error) {
	if req.GetTransactionId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "transaction_id must be positive")
	}

	if req.GetReceiptId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "receipt_id must be positive")
	}

	params := sqlc.SetTransactionReceiptParams{
		ID:        req.GetTransactionId(),
		ReceiptID: req.GetReceiptId(),
	}

	err := s.services.Transactions.SetTransactionReceipt(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.SetTransactionReceiptResponse{
		AffectedRows: 1,
	}, nil
}
