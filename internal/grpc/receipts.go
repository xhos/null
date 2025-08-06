package grpc

import (
	sqlc "ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ==================== RECEIPT SERVICE ====================

func (s *Server) ListReceipts(ctx context.Context, req *pb.ListReceiptsRequest) (*pb.ListReceiptsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	receipts, err := s.services.Receipts.ListForUser(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	pbReceipts := make([]*pb.Receipt, len(receipts))
	for i, receipt := range receipts {
		pbReceipts[i] = toProtoReceipt(&receipt)
	}

	return &pb.ListReceiptsResponse{
		Receipts:   pbReceipts,
		TotalCount: int64(len(receipts)),
	}, nil
}

func (s *Server) GetReceipt(ctx context.Context, req *pb.GetReceiptRequest) (*pb.GetReceiptResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "receipt id must be positive")
	}

	params := sqlc.GetReceiptForUserParams{
		UserID: userID,
		ID:     req.GetId(),
	}

	receipt, err := s.services.Receipts.GetForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// Get receipt items
	items, err := s.services.Receipts.ListItemsForReceipt(ctx, receipt.ID)
	if err != nil {
		return nil, handleError(err)
	}

	pbItems := make([]*pb.ReceiptItem, len(items))
	for i, item := range items {
		pbItems[i] = toProtoReceiptItem(&item)
	}

	pbReceipt := toProtoReceipt(receipt)
	pbReceipt.Items = pbItems

	return &pb.GetReceiptResponse{
		Receipt: pbReceipt,
	}, nil
}

func (s *Server) UploadReceipt(ctx context.Context, req *pb.UploadReceiptRequest) (*pb.UploadReceiptResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if len(req.GetFileData()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "file_data cannot be empty")
	}

	engine := "default"
	if req.Engine != nil {
		engine = req.Engine.String()
	}

	receipt, err := s.services.Receipts.UploadReceipt(ctx, userID, req.GetFileData(), engine)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.UploadReceiptResponse{
		Receipt: toProtoReceipt(receipt),
	}, nil
}

func (s *Server) UpdateReceipt(ctx context.Context, req *pb.UpdateReceiptRequest) (*pb.UpdateReceiptResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "receipt id must be positive")
	}

	// Build update params based on provided fields
	params := sqlc.UpdateReceiptParams{
		ID: req.GetId(),
	}

	// Update notes - not available in current proto, skip for now
	// Update link status and parse status only available fields
	if req.LinkStatus != nil {
		linkStatus := int16(*req.LinkStatus)
		params.LinkStatus = &linkStatus
	}
	if req.ParseStatus != nil {
		parseStatus := int16(*req.ParseStatus)
		params.ParseStatus = &parseStatus
	}

	err = s.services.Receipts.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// Get updated receipt
	getParams := sqlc.GetReceiptForUserParams{
		UserID: userID,
		ID:     req.GetId(),
	}

	receipt, err := s.services.Receipts.GetForUser(ctx, getParams)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.UpdateReceiptResponse{
		Receipt: toProtoReceipt(receipt),
	}, nil
}

func (s *Server) DeleteReceipt(ctx context.Context, req *pb.DeleteReceiptRequest) (*pb.DeleteReceiptResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "receipt id must be positive")
	}

	params := sqlc.DeleteReceiptForUserParams{
		UserID: userID,
		ID:     req.GetId(),
	}

	err = s.services.Receipts.DeleteForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.DeleteReceiptResponse{
		AffectedRows: 1,
	}, nil
}

func (s *Server) ParseReceipt(ctx context.Context, req *pb.ParseReceiptRequest) (*pb.ParseReceiptResponse, error) {
	if req.GetReceiptId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "receipt_id must be positive")
	}

	engine := "default"
	if req.Engine != nil {
		engine = req.Engine.String()
	}

	receipt, err := s.services.Receipts.ParseReceipt(ctx, req.GetReceiptId(), engine)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.ParseReceiptResponse{
		Receipt: toProtoReceipt(receipt),
	}, nil
}

func (s *Server) GetReceiptsByTransaction(ctx context.Context, req *pb.GetReceiptsByTransactionRequest) (*pb.GetReceiptsByTransactionResponse, error) {
	if req.GetTransactionId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "transaction_id must be positive")
	}

	receipts, err := s.services.Receipts.GetReceiptsByTransaction(ctx, req.GetTransactionId())
	if err != nil {
		return nil, handleError(err)
	}

	pbReceipts := make([]*pb.Receipt, len(receipts))
	for i, receipt := range receipts {
		pbReceipts[i] = toProtoReceipt(&receipt)
	}

	return &pb.GetReceiptsByTransactionResponse{
		Receipts: pbReceipts,
	}, nil
}

func (s *Server) SearchReceipts(ctx context.Context, req *pb.SearchReceiptsRequest) (*pb.SearchReceiptsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	if req.GetQuery() == "" {
		return nil, status.Error(codes.InvalidArgument, "query cannot be empty")
	}

	receipts, err := s.services.Receipts.SearchReceipts(ctx, userID, req.GetQuery(), req.Limit)
	if err != nil {
		return nil, handleError(err)
	}

	pbReceipts := make([]*pb.Receipt, len(receipts))
	for i, receipt := range receipts {
		pbReceipts[i] = toProtoReceipt(&receipt)
	}

	return &pb.SearchReceiptsResponse{
		Receipts: pbReceipts,
	}, nil
}

func (s *Server) GetUnlinkedReceipts(ctx context.Context, req *pb.GetUnlinkedReceiptsRequest) (*pb.GetUnlinkedReceiptsResponse, error) {
	receipts, err := s.services.Receipts.GetUnlinked(ctx, req.Limit)
	if err != nil {
		return nil, handleError(err)
	}

	var pbReceipts []*pb.ReceiptSummary
	for _, receipt := range receipts {
		pbReceipts = append(pbReceipts, &pb.ReceiptSummary{
			Id:          receipt.ID,
			Merchant:    receipt.Merchant,
			TotalAmount: receipt.TotalAmount,
			CreatedAt:   toProtoTimestamp(&receipt.CreatedAt),
		})
	}

	return &pb.GetUnlinkedReceiptsResponse{
		Receipts: pbReceipts,
	}, nil
}

func (s *Server) GetReceiptMatchCandidates(ctx context.Context, req *pb.GetReceiptMatchCandidatesRequest) (*pb.GetReceiptMatchCandidatesResponse, error) {
	candidates, err := s.services.Receipts.GetMatchCandidates(ctx)
	if err != nil {
		return nil, handleError(err)
	}

	var pbCandidates []*pb.ReceiptMatchCandidate
	for _, candidate := range candidates {
		// Convert the row to a receipt
		receipt := &sqlc.Receipt{
			ID:           candidate.ID,
			Merchant:     candidate.Merchant,
			PurchaseDate: candidate.PurchaseDate,
			TotalAmount:  candidate.TotalAmount,
			Currency:     candidate.Currency,
		}

		pbCandidates = append(pbCandidates, &pb.ReceiptMatchCandidate{
			Receipt:          toProtoReceipt(receipt),
			PotentialMatches: candidate.PotentialMatches,
		})
	}

	return &pb.GetReceiptMatchCandidatesResponse{
		Candidates: pbCandidates,
	}, nil
}
