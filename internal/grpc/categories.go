package grpc

import (
	sqlc "ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) ListCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListCategoriesResponse, error) {
	categories, err := s.services.Categories.List(ctx)
	if err != nil {
		return nil, handleError(err)
	}

	pbCategories := make([]*pb.Category, len(categories))
	for i, category := range categories {
		pbCategories[i] = toProtoCategory(&category)
	}

	return &pb.ListCategoriesResponse{
		Categories: pbCategories,
		TotalCount: int64(len(categories)),
	}, nil
}

func (s *Server) GetCategory(ctx context.Context, req *pb.GetCategoryRequest) (*pb.GetCategoryResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "category id must be positive")
	}

	category, err := s.services.Categories.Get(ctx, req.GetId())
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetCategoryResponse{
		Category: toProtoCategory(category),
	}, nil
}

func (s *Server) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CreateCategoryResponse, error) {
	if req.GetSlug() == "" {
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}

	if req.GetLabel() == "" {
		return nil, status.Error(codes.InvalidArgument, "label is required")
	}

	if req.GetColor() == "" {
		return nil, status.Error(codes.InvalidArgument, "color is required")
	}

	params := sqlc.CreateCategoryParams{
		Slug:  req.GetSlug(),
		Label: req.GetLabel(),
		Color: req.GetColor(),
	}

	category, err := s.services.Categories.Create(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.CreateCategoryResponse{
		Category: toProtoCategory(category),
	}, nil
}

func (s *Server) UpdateCategory(ctx context.Context, req *pb.UpdateCategoryRequest) (*pb.UpdateCategoryResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "category id must be positive")
	}

	params := sqlc.UpdateCategoryParams{
		ID: req.GetId(),
	}

	if req.Slug != nil {
		params.Slug = req.Slug
	}
	if req.Label != nil {
		params.Label = req.Label
	}
	if req.Color != nil {
		params.Color = req.Color
	}

	category, err := s.services.Categories.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.UpdateCategoryResponse{
		Category: toProtoCategory(category),
	}, nil
}

func (s *Server) DeleteCategory(ctx context.Context, req *pb.DeleteCategoryRequest) (*pb.DeleteCategoryResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "category id must be positive")
	}

	err := s.services.Categories.Delete(ctx, req.GetId())
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.DeleteCategoryResponse{
		AffectedRows: 1,
	}, nil
}

func (s *Server) GetCategoryUsageStats(ctx context.Context, req *pb.GetCategoryUsageStatsRequest) (*pb.GetCategoryUsageStatsResponse, error) {
	if req.GetCategoryId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "category id must be positive")
	}

	category, err := s.services.Categories.Get(ctx, req.GetCategoryId())
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetCategoryUsageStatsResponse{
		Category: toProtoCategory(category),
	}, nil
}

func (s *Server) GetCategoriesWithStats(ctx context.Context, req *pb.GetCategoriesWithStatsRequest) (*pb.GetCategoriesWithStatsResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	categories, err := s.services.Categories.ListForUser(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	pbCategories := make([]*pb.Category, len(categories))
	for i, category := range categories {
		pbCategories[i] = toProtoCategoryFromUserRow(&category)
	}

	return &pb.GetCategoriesWithStatsResponse{
		Categories: pbCategories,
		TotalCount: int64(len(categories)),
	}, nil
}

func (s *Server) SearchCategories(ctx context.Context, req *pb.SearchCategoriesRequest) (*pb.SearchCategoriesResponse, error) {
	if req.GetQuery() == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	// for now, just return all categories and let client filter
	categories, err := s.services.Categories.List(ctx)
	if err != nil {
		return nil, handleError(err)
	}

	pbCategories := make([]*pb.Category, len(categories))
	for i, category := range categories {
		pbCategories[i] = toProtoCategory(&category)
	}

	return &pb.SearchCategoriesResponse{
		Categories: pbCategories,
		TotalCount: int64(len(categories)),
	}, nil
}

func (s *Server) BulkCreateCategories(ctx context.Context, req *pb.BulkCreateCategoriesRequest) (*pb.BulkCreateCategoriesResponse, error) {
	if len(req.Categories) == 0 {
		return nil, status.Error(codes.InvalidArgument, "categories cannot be empty")
	}

	var params []sqlc.BulkCreateCategoriesParams
	for _, category := range req.Categories {
		if category.Slug == "" || category.Label == "" {
			return nil, status.Error(codes.InvalidArgument, "slug and label are required")
		}

		params = append(params, sqlc.BulkCreateCategoriesParams{
			Slug:  category.Slug,
			Label: category.Label,
			Color: category.Color,
		})
	}

	err := s.services.Categories.BulkCreate(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.BulkCreateCategoriesResponse{
		AffectedRows: int64(len(params)),
	}, nil
}

func (s *Server) GetCategoryBySlug(ctx context.Context, req *pb.GetCategoryBySlugRequest) (*pb.GetCategoryBySlugResponse, error) {
	if req.GetSlug() == "" {
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}

	category, err := s.services.Categories.BySlug(ctx, req.GetSlug())
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetCategoryBySlugResponse{
		Category: toProtoCategory(category),
	}, nil
}

func (s *Server) ListCategorySlugs(ctx context.Context, req *pb.ListCategorySlugsRequest) (*pb.ListCategorySlugsResponse, error) {
	slugs, err := s.services.Categories.ListSlugs(ctx)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.ListCategorySlugsResponse{
		Slugs: slugs,
	}, nil
}

func (s *Server) DeleteUnusedCategories(ctx context.Context, req *pb.DeleteUnusedCategoriesRequest) (*pb.DeleteUnusedCategoriesResponse, error) {
	err := s.services.Categories.DeleteUnused(ctx)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.DeleteUnusedCategoriesResponse{
		AffectedRows: 1, // Assuming at least one row affected
	}, nil
}

func (s *Server) GetMostUsedCategoriesForUser(ctx context.Context, req *pb.GetMostUsedCategoriesForUserRequest) (*pb.GetMostUsedCategoriesForUserResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	params := sqlc.GetMostUsedCategoriesForUserParams{
		UserID: userID,
		Limit:  req.Limit,
	}

	categories, err := s.services.Categories.GetMostUsedForUser(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	var pbCategories []*pb.CategoryWithUsage
	for _, cat := range categories {
		pbCategories = append(pbCategories, &pb.CategoryWithUsage{
			Category:    toProtoCategory(&sqlc.Category{ID: cat.ID, Slug: cat.Slug, Label: cat.Label, Color: cat.Color}),
			UsageCount:  cat.UsageCount,
			TotalAmount: &money.Money{CurrencyCode: "CAD", Units: cat.TotalAmount, Nanos: 0},
		})
	}

	return &pb.GetMostUsedCategoriesForUserResponse{
		Categories: pbCategories,
	}, nil
}
