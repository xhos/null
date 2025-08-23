package api

import (
	pb "ariand/internal/gen/arian/v1"
	"context"
	"errors"

	"connectrpc.com/connect"
)

func (s *Server) GetUser(ctx context.Context, req *connect.Request[pb.GetUserRequest]) (*connect.Response[pb.GetUserResponse], error) {
	userID, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}

	user, err := s.services.Users.Get(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetUserResponse{
		User: toProtoUser(user),
	}), nil
}

func (s *Server) GetUserByEmail(ctx context.Context, req *connect.Request[pb.GetUserByEmailRequest]) (*connect.Response[pb.GetUserByEmailResponse], error) {
	user, err := s.services.Users.GetByEmail(ctx, req.Msg.GetEmail())
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetUserByEmailResponse{
		User: toProtoUser(user),
	}), nil
}

func (s *Server) CreateUser(ctx context.Context, req *connect.Request[pb.CreateUserRequest]) (*connect.Response[pb.CreateUserResponse], error) {
	params, err := createUserParamsFromProto(req.Msg)
	if err != nil {
		return nil, err
	}

	user, err := s.services.Users.Create(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.CreateUserResponse{
		User: toProtoUser(user),
	}), nil
}

func (s *Server) UpdateUser(ctx context.Context, req *connect.Request[pb.UpdateUserRequest]) (*connect.Response[pb.UpdateUserResponse], error) {
	params, err := updateUserParamsFromProto(req.Msg)
	if err != nil {
		return nil, err
	}

	user, err := s.services.Users.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateUserResponse{
		User: toProtoUser(user),
	}), nil
}

func (s *Server) DeleteUser(ctx context.Context, req *connect.Request[pb.DeleteUserRequest]) (*connect.Response[pb.DeleteUserResponse], error) {
	userID, err := parseUUID(req.Msg.GetId())
	if err != nil {
		return nil, err
	}

	err = s.services.Users.Delete(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteUserResponse{}), nil
}

func (s *Server) SetUserDefaultAccount(ctx context.Context, req *connect.Request[pb.SetUserDefaultAccountRequest]) (*connect.Response[pb.SetUserDefaultAccountResponse], error) {
	params, err := setUserDefaultAccountParamsFromProto(req.Msg)
	if err != nil {
		return nil, err
	}

	user, err := s.services.Users.SetDefaultAccount(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.SetUserDefaultAccountResponse{
		User: toProtoUser(user),
	}), nil
}

// stub implementations for missing methods required by Connect-Go interface

func (s *Server) UpdateUserDisplayName(ctx context.Context, req *connect.Request[pb.UpdateUserDisplayNameRequest]) (*connect.Response[pb.UpdateUserDisplayNameResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("UpdateUserDisplayName not implemented"))
}

func (s *Server) ListUsers(ctx context.Context, req *connect.Request[pb.ListUsersRequest]) (*connect.Response[pb.ListUsersResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("ListUsers not implemented"))
}

func (s *Server) CheckUserExists(ctx context.Context, req *connect.Request[pb.CheckUserExistsRequest]) (*connect.Response[pb.CheckUserExistsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("CheckUserExists not implemented"))
}
