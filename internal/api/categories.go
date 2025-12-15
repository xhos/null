package api

import (
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"connectrpc.com/connect"
)

// mapSlice transforms a slice using a mapper function
func mapSlice[T any, U any](in []T, f func(*T) *U) []*U {
	out := make([]*U, len(in))
	for i := range in {
		out[i] = f(&in[i])
	}
	return out
}

func (s *Server) ListCategories(ctx context.Context, req *connect.Request[pb.ListCategoriesRequest]) (*connect.Response[pb.ListCategoriesResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	cats, err := s.services.Categories.List(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ListCategoriesResponse{
		Categories: mapSlice(cats, toProtoCategory),
		TotalCount: int64(len(cats)),
	}), nil
}

func (s *Server) GetCategory(ctx context.Context, req *connect.Request[pb.GetCategoryRequest]) (*connect.Response[pb.GetCategoryResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	id := req.Msg.GetId()

	cat, err := s.services.Categories.Get(ctx, userID, id)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetCategoryResponse{
		Category: toProtoCategory(cat),
	}), nil
}

func (s *Server) CreateCategory(ctx context.Context, req *connect.Request[pb.CreateCategoryRequest]) (*connect.Response[pb.CreateCategoryResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.CreateCategoryParams{
		UserID: userID,
		Slug:   req.Msg.GetSlug(),
		Color:  req.Msg.GetColor(),
	}

	cat, err := s.services.Categories.Create(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.CreateCategoryResponse{
		Category: toProtoCategory(cat),
	}), nil
}

func (s *Server) UpdateCategory(ctx context.Context, req *connect.Request[pb.UpdateCategoryRequest]) (*connect.Response[pb.UpdateCategoryResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := sqlc.UpdateCategoryParams{
		ID:     req.Msg.GetId(),
		UserID: userID,
	}

	if req.Msg.Slug != nil {
		params.Slug = req.Msg.Slug
	}
	if req.Msg.Color != nil {
		params.Color = req.Msg.Color
	}

	_, err = s.services.Categories.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateCategoryResponse{}), nil
}

func (s *Server) DeleteCategory(ctx context.Context, req *connect.Request[pb.DeleteCategoryRequest]) (*connect.Response[pb.DeleteCategoryResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	id := req.Msg.GetId()

	affected, err := s.services.Categories.Delete(ctx, userID, id)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteCategoryResponse{
		AffectedRows: affected,
	}), nil
}
