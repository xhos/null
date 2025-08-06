package grpc

import (
	sqlc "ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ==================== ACCOUNT COLLABORATION SERVICE ====================

func (s *Server) AddCollaborator(ctx context.Context, req *pb.AddCollaboratorRequest) (*pb.AddCollaboratorResponse, error) {
	ownerUserID, err := parseUUID(req.GetOwnerUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid owner_user_id: %v", err)
	}

	collaboratorUserID, err := parseUUID(req.GetCollaboratorUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid collaborator_user_id: %v", err)
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account_id must be positive")
	}

	collaborator, err := s.services.Accounts.AddCollaborator(ctx, sqlc.AddAccountCollaboratorParams{
		AccountID:          req.GetAccountId(),
		CollaboratorUserID: collaboratorUserID,
		OwnerUserID:        ownerUserID,
	})
	if err != nil {
		return nil, handleError(err)
	}

	// if no collaborator returned, it might already exist
	if collaborator == nil {
		return nil, status.Error(codes.AlreadyExists, "collaborator already exists")
	}

	// convert AccountUser to AccountCollaborator response
	// Note: we'd need to fetch user details to construct the full User object
	// For now, just return a minimal user with the ID
	user := &pb.User{
		Id: collaborator.UserID.String(),
	}

	pbCollaborator := &pb.AccountCollaborator{
		User:    user,
		AddedAt: toProtoTimestamp(&collaborator.AddedAt),
	}

	return &pb.AddCollaboratorResponse{
		Collaborator: pbCollaborator,
	}, nil
}

func (s *Server) RemoveCollaborator(ctx context.Context, req *pb.RemoveCollaboratorRequest) (*pb.RemoveCollaboratorResponse, error) {
	ownerUserID, err := parseUUID(req.GetOwnerUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid owner_user_id: %v", err)
	}

	collaboratorUserID, err := parseUUID(req.GetCollaboratorUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid collaborator_user_id: %v", err)
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account_id must be positive")
	}

	err = s.services.Accounts.RemoveCollaborator(ctx, sqlc.RemoveAccountCollaboratorParams{
		AccountID:          req.GetAccountId(),
		CollaboratorUserID: collaboratorUserID,
		OwnerUserID:        ownerUserID,
	})
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.RemoveCollaboratorResponse{
		AffectedRows: 1,
	}, nil
}

func (s *Server) ListCollaborators(ctx context.Context, req *pb.ListCollaboratorsRequest) (*pb.ListCollaboratorsResponse, error) {
	requestingUserID, err := parseUUID(req.GetRequestingUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid requesting_user_id: %v", err)
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account_id must be positive")
	}

	collaborators, err := s.services.Accounts.ListCollaborators(ctx, sqlc.ListAccountCollaboratorsParams{
		AccountID:        req.GetAccountId(),
		RequestingUserID: requestingUserID, // check access
	})
	if err != nil {
		return nil, handleError(err)
	}

	pbCollaborators := make([]*pb.AccountCollaborator, len(collaborators))
	for i, collaborator := range collaborators {
		pbCollaborators[i] = toProtoAccountCollaborator(&collaborator)
	}

	return &pb.ListCollaboratorsResponse{
		Collaborators: pbCollaborators,
	}, nil
}

func (s *Server) ListUserCollaborations(ctx context.Context, req *pb.ListUserCollaborationsRequest) (*pb.ListUserCollaborationsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	collaborations, err := s.services.Accounts.ListUserCollaborations(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	pbCollaborations := make([]*pb.AccountCollaboration, len(collaborations))
	for i, collaboration := range collaborations {
		pbCollaborations[i] = toProtoAccountCollaboration(&collaboration)
	}

	return &pb.ListUserCollaborationsResponse{
		Collaborations: pbCollaborations,
	}, nil
}

func (s *Server) LeaveCollaboration(ctx context.Context, req *pb.LeaveCollaborationRequest) (*pb.LeaveCollaborationResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account_id must be positive")
	}

	err = s.services.Accounts.LeaveCollaboration(ctx, sqlc.LeaveAccountCollaborationParams{
		AccountID: req.GetAccountId(),
		UserID:    userID,
	})
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.LeaveCollaborationResponse{
		AffectedRows: 1,
	}, nil
}

func (s *Server) TransferOwnership(ctx context.Context, req *pb.TransferOwnershipRequest) (*pb.TransferOwnershipResponse, error) {
	currentOwnerID, err := parseUUID(req.GetCurrentOwnerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid current_owner_id: %v", err)
	}

	newOwnerID, err := parseUUID(req.GetNewOwnerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid new_owner_id: %v", err)
	}

	if req.GetAccountId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "account_id must be positive")
	}

	// TODO: implement ownership transfer in the service layer
	// This would use TransferAccountOwnershipParams with currentOwnerID, newOwnerID, accountID
	_ = currentOwnerID
	_ = newOwnerID

	return nil, status.Error(codes.Unimplemented, "ownership transfer not yet implemented")
}
