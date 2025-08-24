package api

import (
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"connectrpc.com/connect"
)

func (s *Server) ListTransactions(ctx context.Context, req *connect.Request[pb.ListTransactionsRequest]) (*connect.Response[pb.ListTransactionsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := buildListTransactionsParams(userID, req.Msg)
	transactions, err := s.services.Transactions.ListForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ListTransactionsResponse{
		Transactions: mapSlice(transactions, toProtoTransactionFromListRow),
		TotalCount:   int64(len(transactions)),
		NextCursor:   buildNextCursor(transactions, req.Msg.Limit),
	}), nil
}

func (s *Server) GetTransaction(ctx context.Context, req *connect.Request[pb.GetTransactionRequest]) (*connect.Response[pb.GetTransactionResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.GetTransactionForUserParams{
		UserID: userID,
		ID:     req.Msg.GetId(),
	}

	transaction, err := s.services.Transactions.GetForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetTransactionResponse{
		Transaction: toProtoTransactionFromGetRow(transaction),
	}), nil
}

func (s *Server) CreateTransaction(ctx context.Context, req *connect.Request[pb.CreateTransactionRequest]) (*connect.Response[pb.CreateTransactionResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := buildCreateTransactionParams(userID, req.Msg)
	transactionID, err := s.services.Transactions.CreateForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// get the created transaction
	getParams := sqlc.GetTransactionForUserParams{
		UserID: userID,
		ID:     transactionID,
	}

	transaction, err := s.services.Transactions.GetForUser(ctx, getParams)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.CreateTransactionResponse{
		Transaction: toProtoTransactionFromGetRow(transaction),
	}), nil
}

func (s *Server) UpdateTransaction(ctx context.Context, req *connect.Request[pb.UpdateTransactionRequest]) (*connect.Response[pb.UpdateTransactionResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := buildUpdateTransactionParams(userID, req.Msg)
	err = s.services.Transactions.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// get the updated transaction
	getParams := sqlc.GetTransactionForUserParams{
		UserID: userID,
		ID:     req.Msg.GetId(),
	}

	transaction, err := s.services.Transactions.GetForUser(ctx, getParams)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateTransactionResponse{
		Transaction: toProtoTransactionFromGetRow(transaction),
	}), nil
}

func (s *Server) DeleteTransaction(ctx context.Context, req *connect.Request[pb.DeleteTransactionRequest]) (*connect.Response[pb.DeleteTransactionResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.DeleteTransactionForUserParams{
		UserID: userID,
		ID:     req.Msg.GetId(),
	}

	affectedRows, err := s.services.Transactions.DeleteForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteTransactionResponse{
		AffectedRows: affectedRows,
	}), nil
}

func (s *Server) BulkDeleteTransactions(ctx context.Context, req *connect.Request[pb.BulkDeleteTransactionsRequest]) (*connect.Response[pb.BulkDeleteTransactionsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.BulkDeleteTransactionsForUserParams{
		UserID:         userID,
		TransactionIds: req.Msg.TransactionIds,
	}

	err = s.services.Transactions.BulkDeleteForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.BulkDeleteTransactionsResponse{
		AffectedRows: int64(len(req.Msg.TransactionIds)),
	}), nil
}

func (s *Server) CategorizeTransaction(ctx context.Context, req *connect.Request[pb.CategorizeTransactionRequest]) (*connect.Response[pb.CategorizeTransactionResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	// manual categorization via bulk categorize
	params := sqlc.BulkCategorizeTransactionsForUserParams{
		UserID:         userID,
		TransactionIds: []int64{req.Msg.GetTransactionId()},
		CategoryID:     req.Msg.GetCategoryId(),
	}

	err = s.services.Transactions.BulkCategorizeForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// get the updated transaction
	getParams := sqlc.GetTransactionForUserParams{
		UserID: userID,
		ID:     req.Msg.GetTransactionId(),
	}

	transaction, err := s.services.Transactions.GetForUser(ctx, getParams)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.CategorizeTransactionResponse{
		Transaction: toProtoTransactionFromGetRow(transaction),
	}), nil
}

func (s *Server) SearchTransactions(ctx context.Context, req *connect.Request[pb.SearchTransactionsRequest]) (*connect.Response[pb.SearchTransactionsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	transactions, err := s.services.Transactions.SearchTransactions(ctx, userID, req.Msg.GetQuery(), req.Msg.AccountId, req.Msg.CategoryId, req.Msg.Limit, req.Msg.Offset)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.SearchTransactionsResponse{
		Transactions: mapSlice(transactions, func(tx *sqlc.ListTransactionsForUserRow) *pb.TransactionWithScore {
			return &pb.TransactionWithScore{
				Transaction: toProtoTransactionFromListRow(tx),
			}
		}),
		TotalCount: int64(len(transactions)),
	}), nil
}

func (s *Server) GetTransactionsByAccount(ctx context.Context, req *connect.Request[pb.GetTransactionsByAccountRequest]) (*connect.Response[pb.GetTransactionsByAccountResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	transactions, err := s.services.Transactions.GetTransactionsByAccount(ctx, userID, req.Msg.GetAccountId(), req.Msg.Limit, req.Msg.Offset)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetTransactionsByAccountResponse{
		Transactions: mapSlice(transactions, toProtoTransactionFromListRow),
		TotalCount:   int64(len(transactions)),
		NextCursor:   buildNextCursor(transactions, req.Msg.Limit),
	}), nil
}

func (s *Server) GetUncategorizedTransactions(ctx context.Context, req *connect.Request[pb.GetUncategorizedTransactionsRequest]) (*connect.Response[pb.GetUncategorizedTransactionsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	transactions, err := s.services.Transactions.GetUncategorizedTransactions(ctx, userID, req.Msg.AccountId, req.Msg.Limit, req.Msg.Offset)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetUncategorizedTransactionsResponse{
		Transactions: mapSlice(transactions, toProtoTransactionFromListRow),
		TotalCount:   int64(len(transactions)),
		NextCursor:   buildNextCursor(transactions, req.Msg.Limit),
	}), nil
}

func (s *Server) BulkCategorizeTransactions(ctx context.Context, req *connect.Request[pb.BulkCategorizeTransactionsRequest]) (*connect.Response[pb.BulkCategorizeTransactionsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.BulkCategorizeTransactionsForUserParams{
		UserID:         userID,
		TransactionIds: req.Msg.TransactionIds,
		CategoryID:     req.Msg.GetCategoryId(),
	}

	err = s.services.Transactions.BulkCategorizeForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.BulkCategorizeTransactionsResponse{
		AffectedRows: int64(len(req.Msg.TransactionIds)),
	}), nil
}

func (s *Server) GetTransactionCountByAccount(ctx context.Context, req *connect.Request[pb.GetTransactionCountByAccountRequest]) (*connect.Response[pb.GetTransactionCountByAccountResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	counts, err := s.services.Transactions.GetTransactionCountByAccountForUser(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetTransactionCountByAccountResponse{
		Counts: mapSlice(counts, func(count *sqlc.GetTransactionCountByAccountForUserRow) *pb.TransactionCountByAccount {
			return &pb.TransactionCountByAccount{
				AccountId:        count.ID,
				TransactionCount: count.TransactionCount,
			}
		}),
	}), nil
}

func (s *Server) FindCandidateTransactions(ctx context.Context, req *connect.Request[pb.FindCandidateTransactionsRequest]) (*connect.Response[pb.FindCandidateTransactionsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.FindCandidateTransactionsForUserParams{
		UserID:   userID,
		Merchant: req.Msg.GetMerchant(),
		Date:     timestampToDate(req.Msg.PurchaseDate),
		Total:    moneyToDecimal(req.Msg.TotalAmount),
	}

	candidates, err := s.services.Transactions.FindCandidateTransactionsForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.FindCandidateTransactionsResponse{
		Candidates: mapSlice(candidates, func(candidate *sqlc.FindCandidateTransactionsForUserRow) *pb.TransactionWithScore {
			return &pb.TransactionWithScore{
				Transaction: toProtoTransactionFromFindRow(candidate),
			}
		}),
	}), nil
}

func (s *Server) IdentifyMerchant(ctx context.Context, req *connect.Request[pb.IdentifyMerchantRequest]) (*connect.Response[pb.IdentifyMerchantResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	err = s.services.Transactions.IdentifyMerchantForTransaction(ctx, userID, req.Msg.GetTransactionId())
	if err != nil {
		return nil, handleError(err)
	}

	// get the updated transaction to return the identified merchant
	getParams := sqlc.GetTransactionForUserParams{
		UserID: userID,
		ID:     req.Msg.GetTransactionId(),
	}

	transaction, err := s.services.Transactions.GetForUser(ctx, getParams)
	if err != nil {
		return nil, handleError(err)
	}

	merchant := ""
	if transaction.Merchant != nil {
		merchant = *transaction.Merchant
	}

	return connect.NewResponse(&pb.IdentifyMerchantResponse{
		Merchant: merchant,
	}), nil
}

func (s *Server) SetTransactionReceipt(ctx context.Context, req *connect.Request[pb.SetTransactionReceiptRequest]) (*connect.Response[pb.SetTransactionReceiptResponse], error) {
	params := sqlc.SetTransactionReceiptParams{
		ID:        req.Msg.GetTransactionId(),
		ReceiptID: req.Msg.GetReceiptId(),
	}

	err := s.services.Transactions.SetTransactionReceipt(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.SetTransactionReceiptResponse{
		AffectedRows: 1,
	}), nil
}
