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
	transactions, err := s.services.Transactions.List(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ListTransactionsResponse{
		Transactions: mapSlice(transactions, toProtoTransaction),
		TotalCount:   int64(len(transactions)),
		NextCursor:   buildNextCursor(transactions, req.Msg.Limit),
	}), nil
}

func (s *Server) GetTransaction(ctx context.Context, req *connect.Request[pb.GetTransactionRequest]) (*connect.Response[pb.GetTransactionResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.GetTransactionParams{
		UserID: userID,
		ID:     req.Msg.GetId(),
	}

	transaction, err := s.services.Transactions.Get(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetTransactionResponse{
		Transaction: convertTransactionToProto(transaction),
	}), nil
}

func (s *Server) CreateTransaction(ctx context.Context, req *connect.Request[pb.CreateTransactionRequest]) (*connect.Response[pb.CreateTransactionResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	paramsList, err := buildCreateTransactionParamsList(userID, req.Msg)
	if err != nil {
		return nil, handleError(err)
	}

	transactions, err := s.services.Transactions.Create(ctx, userID, paramsList)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.CreateTransactionResponse{
		Transactions: mapSlice(transactions, toProtoTransaction),
		CreatedCount: int32(len(transactions)),
	}), nil
}

func (s *Server) UpdateTransaction(ctx context.Context, req *connect.Request[pb.UpdateTransactionRequest]) (*connect.Response[pb.UpdateTransactionResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params, err := buildUpdateTransactionParams(userID, req.Msg)
	if err != nil {
		return nil, handleError(err)
	}
	err = s.services.Transactions.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateTransactionResponse{}), nil
}

func (s *Server) DeleteTransaction(ctx context.Context, req *connect.Request[pb.DeleteTransactionRequest]) (*connect.Response[pb.DeleteTransactionResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.BulkDeleteTransactionsParams{
		UserID:         userID,
		TransactionIds: req.Msg.Ids,
	}

	err = s.services.Transactions.BulkDelete(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteTransactionResponse{
		AffectedRows: int64(len(req.Msg.Ids)),
	}), nil
}

func (s *Server) CategorizeTransactions(ctx context.Context, req *connect.Request[pb.CategorizeTransactionsRequest]) (*connect.Response[pb.CategorizeTransactionsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.BulkCategorizeTransactionsParams{
		UserID:         userID,
		TransactionIds: req.Msg.TransactionIds,
		CategoryID:     req.Msg.GetCategoryId(),
	}

	err = s.services.Transactions.Categorize(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.CategorizeTransactionsResponse{
		AffectedRows: int64(len(req.Msg.TransactionIds)),
	}), nil
}
