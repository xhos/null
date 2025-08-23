package api

import (
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"
	"errors"

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
	cats, err := s.services.Categories.List(ctx)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ListCategoriesResponse{
		Categories: mapSlice(cats, toProtoCategory),
		TotalCount: int64(len(cats)),
	}), nil
}

func (s *Server) GetCategory(ctx context.Context, req *connect.Request[pb.GetCategoryRequest]) (*connect.Response[pb.GetCategoryResponse], error) {
	id := req.Msg.GetId()

	cat, err := s.services.Categories.Get(ctx, id)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetCategoryResponse{
		Category: toProtoCategory(cat),
	}), nil
}

func (s *Server) CreateCategory(ctx context.Context, req *connect.Request[pb.CreateCategoryRequest]) (*connect.Response[pb.CreateCategoryResponse], error) {
	params := sqlc.CreateCategoryParams{
		Slug:  req.Msg.GetSlug(),
		Label: req.Msg.GetLabel(),
		Color: req.Msg.GetColor(),
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
	params := sqlc.UpdateCategoryParams{
		ID: req.Msg.GetId(),
	}

	if req.Msg.Slug != nil {
		params.Slug = req.Msg.Slug
	}
	if req.Msg.Label != nil {
		params.Label = req.Msg.Label
	}
	if req.Msg.Color != nil {
		params.Color = req.Msg.Color
	}

	cat, err := s.services.Categories.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateCategoryResponse{
		Category: toProtoCategory(cat),
	}), nil
}

func (s *Server) DeleteCategory(ctx context.Context, req *connect.Request[pb.DeleteCategoryRequest]) (*connect.Response[pb.DeleteCategoryResponse], error) {
	id := req.Msg.GetId()

	affected, err := s.services.Categories.Delete(ctx, id)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteCategoryResponse{
		AffectedRows: affected,
	}), nil
}

func (s *Server) GetCategoryBySlug(ctx context.Context, req *connect.Request[pb.GetCategoryBySlugRequest]) (*connect.Response[pb.GetCategoryBySlugResponse], error) {
	slug := req.Msg.GetSlug()

	cat, err := s.services.Categories.BySlug(ctx, slug)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetCategoryBySlugResponse{
		Category: toProtoCategory(cat),
	}), nil
}

func (s *Server) BulkCreateCategories(ctx context.Context, req *connect.Request[pb.BulkCreateCategoriesRequest]) (*connect.Response[pb.BulkCreateCategoriesResponse], error) {
	categories := make([]*pb.Category, 0, len(req.Msg.Categories))
	
	for _, catReq := range req.Msg.Categories {
		params := sqlc.CreateCategoryParams{
			Slug:  catReq.GetSlug(),
			Label: catReq.GetLabel(),
			Color: catReq.GetColor(),
		}

		cat, err := s.services.Categories.Create(ctx, params)
		if err != nil {
			return nil, handleError(err)
		}

		categories = append(categories, toProtoCategory(cat))
	}

	return connect.NewResponse(&pb.BulkCreateCategoriesResponse{
		Categories:   categories,
		AffectedRows: int64(len(categories)),
	}), nil
}

func (s *Server) ListCategoriesWithUsage(ctx context.Context, req *connect.Request[pb.ListCategoriesWithUsageRequest]) (*connect.Response[pb.ListCategoriesWithUsageResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("ListCategoriesWithUsage not implemented"))
}

func (s *Server) ListCategoriesForUser(ctx context.Context, req *connect.Request[pb.ListCategoriesForUserRequest]) (*connect.Response[pb.ListCategoriesForUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("ListCategoriesForUser not implemented"))
}

func (s *Server) GetCategoryWithStats(ctx context.Context, req *connect.Request[pb.GetCategoryWithStatsRequest]) (*connect.Response[pb.GetCategoryWithStatsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("GetCategoryWithStats not implemented"))
}

func (s *Server) DeleteUnusedCategories(ctx context.Context, req *connect.Request[pb.DeleteUnusedCategoriesRequest]) (*connect.Response[pb.DeleteUnusedCategoriesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("DeleteUnusedCategories not implemented"))
}

func (s *Server) GetCategoryUsageStats(ctx context.Context, req *connect.Request[pb.GetCategoryUsageStatsRequest]) (*connect.Response[pb.GetCategoryUsageStatsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("GetCategoryUsageStats not implemented"))
}

func (s *Server) GetCategoriesWithStats(ctx context.Context, req *connect.Request[pb.GetCategoriesWithStatsRequest]) (*connect.Response[pb.GetCategoriesWithStatsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("GetCategoriesWithStats not implemented"))
}

func (s *Server) SearchCategories(ctx context.Context, req *connect.Request[pb.SearchCategoriesRequest]) (*connect.Response[pb.SearchCategoriesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("SearchCategories not implemented"))
}

func (s *Server) GetMostUsedCategoriesForUser(ctx context.Context, req *connect.Request[pb.GetMostUsedCategoriesForUserRequest]) (*connect.Response[pb.GetMostUsedCategoriesForUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("GetMostUsedCategoriesForUser not implemented"))
}

func (s *Server) GetUnusedCategories(ctx context.Context, req *connect.Request[pb.GetUnusedCategoriesRequest]) (*connect.Response[pb.GetUnusedCategoriesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("GetUnusedCategories not implemented"))
}

func (s *Server) ListCategorySlugs(ctx context.Context, req *connect.Request[pb.ListCategorySlugsRequest]) (*connect.Response[pb.ListCategorySlugsResponse], error) {
	slugs, err := s.services.Categories.ListSlugs(ctx)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ListCategorySlugsResponse{
		Slugs: slugs,
	}), nil
}
