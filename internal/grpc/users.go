package grpc

import (
	pb "ariand/internal/gen/arian/v1"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TODO: List Users not implemented yet

// ==================== USER SERVICE ====================

func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	userID, err := parseUUID(req.GetId())
	if err != nil {
		return nil, err
	}

	user, err := s.services.Users.Get(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetUserResponse{
		User: toProtoUser(user),
	}, nil
}

func (s *Server) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.GetUserByEmailResponse, error) {
	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	user, err := s.services.Users.GetByEmail(ctx, req.GetEmail())
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetUserByEmailResponse{
		User: toProtoUser(user),
	}, nil
}

func (s *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	params := createUserParamsFromProto(req)
	user, err := s.services.Users.Create(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.CreateUserResponse{
		User: toProtoUser(user),
	}, nil
}

func (s *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	params, err := updateUserParamsFromProto(req)
	if err != nil {
		return nil, err
	}

	user, err := s.services.Users.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.UpdateUserResponse{
		User: toProtoUser(user),
	}, nil
}

func (s *Server) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	userID, err := parseUUID(req.GetId())
	if err != nil {
		return nil, err
	}

	err = s.services.Users.Delete(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.DeleteUserResponse{}, nil
}

func (s *Server) SetUserDefaultAccount(ctx context.Context, req *pb.SetUserDefaultAccountRequest) (*pb.SetUserDefaultAccountResponse, error) {
	params, err := setUserDefaultAccountParamsFromProto(req)
	if err != nil {
		return nil, err
	}

	user, err := s.services.Users.SetDefaultAccount(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.SetUserDefaultAccountResponse{
		User: toProtoUser(user),
	}, nil
}
