package api

import (
	pb "ariand/internal/gen/arian/v1"
	"context"

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

	err = s.services.Users.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateUserResponse{}), nil
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
