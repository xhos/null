package api

import (
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"connectrpc.com/connect"
)

func (s *Server) ListReceipts(ctx context.Context, req *connect.Request[pb.ListReceiptsRequest]) (*connect.Response[pb.ListReceiptsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	receipts, err := s.services.Receipts.ListForUser(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ListReceiptsResponse{
		Receipts:   mapSlice(receipts, toProtoReceipt),
		TotalCount: int64(len(receipts)),
	}), nil
}

func (s *Server) GetReceipt(ctx context.Context, req *connect.Request[pb.GetReceiptRequest]) (*connect.Response[pb.GetReceiptResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.GetReceiptForUserParams{
		UserID: userID,
		ID:     req.Msg.GetId(),
	}

	receipt, err := s.services.Receipts.GetForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// get receipt items
	items, err := s.services.Receipts.ListItemsForReceipt(ctx, receipt.ID)
	if err != nil {
		return nil, handleError(err)
	}

	pbReceipt := toProtoReceipt(receipt)
	pbReceipt.Items = buildReceiptItemsResponse(items)

	return connect.NewResponse(&pb.GetReceiptResponse{
		Receipt: pbReceipt,
	}), nil
}

func (s *Server) UploadReceipt(ctx context.Context, req *connect.Request[pb.UploadReceiptRequest]) (*connect.Response[pb.UploadReceiptResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	engine := "default"
	if req.Msg.Engine != nil {
		engine = req.Msg.Engine.String()
	}

	receipt, err := s.services.Receipts.UploadReceipt(ctx, userID, req.Msg.GetFileData(), engine)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UploadReceiptResponse{
		Receipt: toProtoReceipt(receipt),
	}), nil
}

func (s *Server) UpdateReceipt(ctx context.Context, req *connect.Request[pb.UpdateReceiptRequest]) (*connect.Response[pb.UpdateReceiptResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.UpdateReceiptParams{
		ID: req.Msg.GetId(),
	}

	// update available fields
	if req.Msg.LinkStatus != nil {
		linkStatus := int16(*req.Msg.LinkStatus)
		params.LinkStatus = &linkStatus
	}
	if req.Msg.ParseStatus != nil {
		parseStatus := int16(*req.Msg.ParseStatus)
		params.ParseStatus = &parseStatus
	}

	err = s.services.Receipts.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// get updated receipt
	getParams := sqlc.GetReceiptForUserParams{
		UserID: userID,
		ID:     req.Msg.GetId(),
	}

	receipt, err := s.services.Receipts.GetForUser(ctx, getParams)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateReceiptResponse{
		Receipt: toProtoReceipt(receipt),
	}), nil
}

func (s *Server) DeleteReceipt(ctx context.Context, req *connect.Request[pb.DeleteReceiptRequest]) (*connect.Response[pb.DeleteReceiptResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.DeleteReceiptForUserParams{
		UserID: userID,
		ID:     req.Msg.GetId(),
	}

	err = s.services.Receipts.DeleteForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteReceiptResponse{
		AffectedRows: 1,
	}), nil
}

func (s *Server) ParseReceipt(ctx context.Context, req *connect.Request[pb.ParseReceiptRequest]) (*connect.Response[pb.ParseReceiptResponse], error) {
	engine := "default"
	if req.Msg.Engine != nil {
		engine = req.Msg.Engine.String()
	}

	receipt, err := s.services.Receipts.ParseReceipt(ctx, req.Msg.GetReceiptId(), engine)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ParseReceiptResponse{
		Receipt: toProtoReceipt(receipt),
	}), nil
}

func (s *Server) GetReceiptsByTransaction(ctx context.Context, req *connect.Request[pb.GetReceiptsByTransactionRequest]) (*connect.Response[pb.GetReceiptsByTransactionResponse], error) {
	receipts, err := s.services.Receipts.GetReceiptsByTransaction(ctx, req.Msg.GetTransactionId())
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetReceiptsByTransactionResponse{
		Receipts: mapSlice(receipts, toProtoReceipt),
	}), nil
}

func (s *Server) SearchReceipts(ctx context.Context, req *connect.Request[pb.SearchReceiptsRequest]) (*connect.Response[pb.SearchReceiptsResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	receipts, err := s.services.Receipts.SearchReceipts(ctx, userID, req.Msg.GetQuery(), req.Msg.Limit)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.SearchReceiptsResponse{
		Receipts: mapSlice(receipts, toProtoReceipt),
	}), nil
}

func (s *Server) GetUnlinkedReceipts(ctx context.Context, req *connect.Request[pb.GetUnlinkedReceiptsRequest]) (*connect.Response[pb.GetUnlinkedReceiptsResponse], error) {
	receipts, err := s.services.Receipts.GetUnlinked(ctx, req.Msg.Limit)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetUnlinkedReceiptsResponse{
		Receipts: buildReceiptSummariesResponse(receipts),
	}), nil
}

func (s *Server) GetReceiptMatchCandidates(ctx context.Context, req *connect.Request[pb.GetReceiptMatchCandidatesRequest]) (*connect.Response[pb.GetReceiptMatchCandidatesResponse], error) {
	candidates, err := s.services.Receipts.GetMatchCandidates(ctx)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetReceiptMatchCandidatesResponse{
		Candidates: buildMatchCandidatesResponse(candidates),
	}), nil
}

func (s *Server) BulkCreateReceiptItems(ctx context.Context, req *connect.Request[pb.BulkCreateReceiptItemsRequest]) (*connect.Response[pb.BulkCreateReceiptItemsResponse], error) {
	params := buildBulkCreateReceiptItemsParams(req.Msg.GetItems())

	err := s.services.Receipts.BulkCreateItems(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// return placeholder items - in real implementation would fetch created items
	pbItems := make([]*pb.ReceiptItem, len(req.Msg.GetItems()))
	for i, item := range req.Msg.GetItems() {
		pbItems[i] = &pb.ReceiptItem{
			Id:        0, // would be set by database
			ReceiptId: item.GetReceiptId(),
			Name:      item.GetName(),
			LineNo:    item.LineNo,
			Quantity:  1.0,
		}
	}

	return connect.NewResponse(&pb.BulkCreateReceiptItemsResponse{
		Items:        pbItems,
		AffectedRows: int64(len(req.Msg.GetItems())),
	}), nil
}

func (s *Server) CreateReceipt(ctx context.Context, req *connect.Request[pb.CreateReceiptRequest]) (*connect.Response[pb.CreateReceiptResponse], error) {
	params := sqlc.CreateReceiptParams{
		Engine:   int16(req.Msg.Engine),
		Merchant: req.Msg.Merchant,
	}

	receipt, err := s.services.Receipts.Create(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.CreateReceiptResponse{
		Receipt: toProtoReceipt(receipt),
	}), nil
}

func (s *Server) CreateReceiptItem(ctx context.Context, req *connect.Request[pb.CreateReceiptItemRequest]) (*connect.Response[pb.CreateReceiptItemResponse], error) {
	params := buildReceiptItemParams(req.Msg)
	item, err := s.services.Receipts.CreateItem(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.CreateReceiptItemResponse{
		Item: toProtoReceiptItem(item),
	}), nil
}

func (s *Server) DeleteReceiptItem(ctx context.Context, req *connect.Request[pb.DeleteReceiptItemRequest]) (*connect.Response[pb.DeleteReceiptItemResponse], error) {
	// the service method doesn't return affected rows, so assume 1 for success
	err := s.services.Receipts.DeleteItemsByReceipt(ctx, req.Msg.GetId())
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteReceiptItemResponse{
		AffectedRows: 1,
	}), nil
}

func (s *Server) DeleteReceiptItemsByReceipt(ctx context.Context, req *connect.Request[pb.DeleteReceiptItemsByReceiptRequest]) (*connect.Response[pb.DeleteReceiptItemsByReceiptResponse], error) {
	err := s.services.Receipts.DeleteItemsByReceipt(ctx, req.Msg.GetReceiptId())
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteReceiptItemsByReceiptResponse{
		AffectedRows: 1, // placeholder since service doesn't return count
	}), nil
}

func (s *Server) GetReceiptItem(ctx context.Context, req *connect.Request[pb.GetReceiptItemRequest]) (*connect.Response[pb.GetReceiptItemResponse], error) {
	item, err := s.services.Receipts.GetItem(ctx, req.Msg.GetId())
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetReceiptItemResponse{
		Item: toProtoReceiptItem(item),
	}), nil
}

func (s *Server) ListReceiptItems(ctx context.Context, req *connect.Request[pb.ListReceiptItemsRequest]) (*connect.Response[pb.ListReceiptItemsResponse], error) {
	items, err := s.services.Receipts.ListItemsForReceipt(ctx, req.Msg.GetReceiptId())
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ListReceiptItemsResponse{
		Items: buildReceiptItemsResponse(items),
	}), nil
}

func (s *Server) UpdateReceiptItem(ctx context.Context, req *connect.Request[pb.UpdateReceiptItemRequest]) (*connect.Response[pb.UpdateReceiptItemResponse], error) {
	params := buildUpdateReceiptItemParams(req.Msg)
	item, err := s.services.Receipts.UpdateItem(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateReceiptItemResponse{
		Item: toProtoReceiptItem(item),
	}), nil
}
