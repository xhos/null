package service

import (
	"context"
	"fmt"
	"strings"

	"null/internal/db/sqlc"
	pb "null/internal/gen/null/v1"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// ----- interface ---------------------------------------------------------------------------

type UserService interface {
	Create(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error)
	Get(ctx context.Context, id string) (*pb.User, error)
	Update(ctx context.Context, req *pb.UpdateUserRequest) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*pb.User, error)
}

type userSvc struct {
	queries *sqlc.Queries
	log     *log.Logger
}

func newUserSvc(queries *sqlc.Queries, logger *log.Logger) UserService {
	return &userSvc{queries: queries, log: logger}
}

// ----- methods -----------------------------------------------------------------------------

func (s *userSvc) Create(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	params, err := buildCreateUserParams(req)
	if err != nil {
		return nil, wrapErr("UserService.Create", err)
	}

	params.Email = strings.ToLower(params.Email)

	user, err := s.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, wrapErr("UserService.Create", err)
	}

	return userToPb(&user), nil
}

func (s *userSvc) Get(ctx context.Context, id string) (*pb.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, wrapErr("UserService.Get", fmt.Errorf("invalid user_id: %w", err))
	}

	user, err := s.queries.GetUser(ctx, userID)
	if err != nil {
		return nil, wrapErr("UserService.Get", err)
	}

	return userToPb(&user), nil
}

func (s *userSvc) Update(ctx context.Context, req *pb.UpdateUserRequest) error {
	params, err := buildUpdateUserParams(req)
	if err != nil {
		return wrapErr("UserService.Update", err)
	}

	if params.Email != nil {
		normalized := strings.ToLower(*params.Email)
		params.Email = &normalized
	}

	err = s.queries.UpdateUser(ctx, params)
	if err != nil {
		return wrapErr("UserService.Update", err)
	}

	return nil
}

func (s *userSvc) Delete(ctx context.Context, id string) error {
	userID, err := uuid.Parse(id)
	if err != nil {
		return wrapErr("UserService.Delete", fmt.Errorf("invalid user_id: %w", err))
	}

	rowsAffected, err := s.queries.DeleteUserWithCascade(ctx, userID)
	if err != nil {
		return wrapErr("UserService.Delete", err)
	}

	s.log.Debug("user deleted with cascade", "user_id", userID, "rows_affected", rowsAffected)

	return nil
}

func (s *userSvc) List(ctx context.Context) ([]*pb.User, error) {
	users, err := s.queries.ListUsers(ctx)
	if err != nil {
		return nil, wrapErr("UserService.List", err)
	}

	pbUsers := make([]*pb.User, len(users))
	for i, user := range users {
		u := user
		pbUsers[i] = userToPb(&u)
	}

	return pbUsers, nil
}

// ----- param builders ----------------------------------------------------------------------

func buildCreateUserParams(req *pb.CreateUserRequest) (sqlc.CreateUserParams, error) {
	userID, err := uuid.Parse(req.GetId())
	if err != nil {
		return sqlc.CreateUserParams{}, fmt.Errorf("invalid user_id: %w", err)
	}

	return sqlc.CreateUserParams{
		ID:          userID,
		Email:       req.GetEmail(),
		DisplayName: req.DisplayName,
	}, nil
}

func buildUpdateUserParams(req *pb.UpdateUserRequest) (sqlc.UpdateUserParams, error) {
	userID, err := uuid.Parse(req.GetId())
	if err != nil {
		return sqlc.UpdateUserParams{}, fmt.Errorf("invalid user_id: %w", err)
	}

	return sqlc.UpdateUserParams{
		ID:              userID,
		Email:           req.Email,
		DisplayName:     req.DisplayName,
		PrimaryCurrency: req.PrimaryCurrency,
		Timezone:        req.Timezone,
	}, nil
}

// ----- conversion helpers ------------------------------------------------------------------

func userToPb(u *sqlc.User) *pb.User {
	if u == nil {
		return nil
	}

	return &pb.User{
		Id:              u.ID.String(),
		Email:           u.Email,
		DisplayName:     u.DisplayName,
		PrimaryCurrency: u.PrimaryCurrency,
		Timezone:        u.Timezone,
		CreatedAt:       toProtoTimestamp(&u.CreatedAt),
		UpdatedAt:       toProtoTimestamp(&u.UpdatedAt),
	}
}
