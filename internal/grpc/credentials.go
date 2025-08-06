package grpc

import (
	sqlc "ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ==================== CREDENTIAL SERVICE ====================

func (s *Server) ListCredentials(ctx context.Context, req *pb.ListCredentialsRequest) (*pb.ListCredentialsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	credentials, err := s.services.Auth.ListCredentialsByUser(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	pbCredentials := make([]*pb.UserCredential, len(credentials))
	for i, credential := range credentials {
		pbCredentials[i] = toProtoUserCredential(&credential)
	}

	return &pb.ListCredentialsResponse{
		Credentials: pbCredentials,
	}, nil
}

func (s *Server) GetCredential(ctx context.Context, req *pb.GetCredentialRequest) (*pb.GetCredentialResponse, error) {
	credentialID, err := parseUUID(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid credential id: %v", err)
	}

	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	credential, err := s.services.Auth.GetCredentialForUser(ctx, sqlc.GetCredentialForUserParams{
		ID:     credentialID,
		UserID: userID,
	})
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetCredentialResponse{
		Credential: toProtoUserCredential(credential),
	}, nil
}

func (s *Server) CreateCredential(ctx context.Context, req *pb.CreateCredentialRequest) (*pb.CreateCredentialResponse, error) {
	if len(req.GetCredentialId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "credential_id is required")
	}
	if len(req.GetPublicKey()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "public_key is required")
	}

	params, err := createCredentialParamsFromProto(req)
	if err != nil {
		return nil, err
	}

	credential, err := s.services.Auth.CreateCredential(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.CreateCredentialResponse{
		Credential: toProtoUserCredential(credential),
	}, nil
}

func (s *Server) DeleteCredential(ctx context.Context, req *pb.DeleteCredentialRequest) (*pb.DeleteCredentialResponse, error) {
	credentialID, err := parseUUID(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid credential id: %v", err)
	}

	callerID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	err = s.services.Auth.DeleteCredentialForUser(ctx, sqlc.DeleteCredentialForUserParams{
		ID:     credentialID,
		UserID: callerID,
	}, callerID)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.DeleteCredentialResponse{}, nil
}

func (s *Server) GetCredentialByCredentialID(ctx context.Context, req *pb.GetCredentialByCredentialIDRequest) (*pb.GetCredentialByCredentialIDResponse, error) {
	if len(req.GetCredentialId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "credential_id cannot be empty")
	}

	credential, err := s.services.Auth.GetCredentialByCredentialID(ctx, req.GetCredentialId())
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetCredentialByCredentialIDResponse{
		Credential: toProtoUserCredential(credential),
	}, nil
}

func (s *Server) GetCredentialForUser(ctx context.Context, req *pb.GetCredentialForUserRequest) (*pb.GetCredentialForUserResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	credentialID, err := parseUUID(req.GetId())
	if err != nil {
		return nil, err
	}

	params := sqlc.GetCredentialForUserParams{
		UserID: userID,
		ID:     credentialID,
	}

	credential, err := s.services.Auth.GetCredentialForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetCredentialForUserResponse{
		Credential: toProtoUserCredential(credential),
	}, nil
}

func (s *Server) UpdateCredentialSignCountByCredentialID(ctx context.Context, req *pb.UpdateCredentialSignCountByCredentialIDRequest) (*pb.UpdateCredentialSignCountByCredentialIDResponse, error) {
	if len(req.GetCredentialId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "credential_id cannot be empty")
	}

	params := sqlc.UpdateCredentialSignCountByCredentialIdParams{
		CredentialID: req.GetCredentialId(),
		SignCount:    req.GetSignCount(),
	}

	err := s.services.Auth.UpdateCredentialSignCountByCredentialID(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.UpdateCredentialSignCountByCredentialIDResponse{}, nil
}
