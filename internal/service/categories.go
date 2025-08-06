package service

import (
	sqlc "ariand/internal/db/sqlc"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

type CategoryService interface {
	List(ctx context.Context) ([]sqlc.Category, error)
	ListWithUsage(ctx context.Context, userID uuid.UUID, startDate *string, endDate *string, limit *int32) ([]sqlc.ListCategoriesWithUsageRow, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]sqlc.ListCategoriesForUserRow, error)
	Get(ctx context.Context, id int64) (*sqlc.Category, error)
	GetWithStats(ctx context.Context, userID uuid.UUID, id int64, startDate *string, endDate *string) (*sqlc.GetCategoryWithStatsRow, error)
	Create(ctx context.Context, params sqlc.CreateCategoryParams) (*sqlc.Category, error)
	BulkCreate(ctx context.Context, categories []sqlc.BulkCreateCategoriesParams) error
	Update(ctx context.Context, params sqlc.UpdateCategoryParams) (*sqlc.Category, error)
	Delete(ctx context.Context, id int64) error
	DeleteUnused(ctx context.Context) error
	BySlug(ctx context.Context, slug string) (*sqlc.Category, error)
	ListSlugs(ctx context.Context) ([]string, error)
	Search(ctx context.Context, query string) ([]sqlc.Category, error)
	GetMostUsedForUser(ctx context.Context, params sqlc.GetMostUsedCategoriesForUserParams) ([]sqlc.GetMostUsedCategoriesForUserRow, error)
	GetUnused(ctx context.Context) ([]sqlc.Category, error)
	GetCategoryUsageStats(ctx context.Context, userID uuid.UUID, id int64, startDate *string, endDate *string) (*sqlc.GetCategoryWithStatsRow, error)
	GetCategoriesWithStats(ctx context.Context, userID uuid.UUID, startDate *string, endDate *string, limit *int32) ([]sqlc.ListCategoriesWithUsageRow, error)
}

var (
	slugCache   []string
	slugCacheMu sync.RWMutex
	slugRegex   = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9]+)*$`)
)

type catSvc struct {
	queries *sqlc.Queries
	log     *log.Logger
}

func newCatSvc(queries *sqlc.Queries, lg *log.Logger) CategoryService {
	return &catSvc{queries: queries, log: lg}
}

func (s *catSvc) List(ctx context.Context) ([]sqlc.Category, error) {
	categories, err := s.queries.ListCategories(ctx)
	if err != nil {
		return nil, wrapErr("CategoryService.List", err)
	}
	return categories, nil
}

func (s *catSvc) ListForUser(ctx context.Context, userID uuid.UUID) ([]sqlc.ListCategoriesForUserRow, error) {
	categories, err := s.queries.ListCategoriesForUser(ctx, userID)
	if err != nil {
		return nil, wrapErr("CategoryService.ListForUser", err)
	}
	return categories, nil
}

func (s *catSvc) Get(ctx context.Context, id int64) (*sqlc.Category, error) {
	category, err := s.queries.GetCategory(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("CategoryService.Get", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("CategoryService.Get", err)
	}
	return &category, nil
}

func (s *catSvc) Create(ctx context.Context, params sqlc.CreateCategoryParams) (*sqlc.Category, error) {
	if !slugRegex.MatchString(params.Slug) {
		return nil, wrapErr("CategoryService.Create", ErrValidation)
	}

	category, err := s.queries.CreateCategory(ctx, params)
	if err != nil {
		return nil, wrapErr("CategoryService.Create", err)
	}

	s.invalidateSlugCache()

	return &category, nil
}

func (s *catSvc) Update(ctx context.Context, params sqlc.UpdateCategoryParams) (*sqlc.Category, error) {
	if params.Slug != nil && !slugRegex.MatchString(*params.Slug) {
		return nil, wrapErr("CategoryService.Update", ErrValidation)
	}

	category, err := s.queries.UpdateCategory(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("CategoryService.Update", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("CategoryService.Update", err)
	}

	s.invalidateSlugCache()

	return &category, nil
}

func (s *catSvc) Delete(ctx context.Context, id int64) error {
	_, err := s.queries.DeleteCategory(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return wrapErr("CategoryService.Delete", ErrNotFound)
	}
	if err != nil {
		return wrapErr("CategoryService.Delete", err)
	}

	s.invalidateSlugCache()

	return nil
}

func (s *catSvc) BySlug(ctx context.Context, slug string) (*sqlc.Category, error) {
	category, err := s.queries.GetCategoryBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("CategoryService.BySlug", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("CategoryService.BySlug", err)
	}
	return &category, nil
}

func (s *catSvc) ListSlugs(ctx context.Context) ([]string, error) {
	slugCacheMu.RLock()
	if slugCache != nil {
		cached := make([]string, len(slugCache))
		copy(cached, slugCache)
		slugCacheMu.RUnlock()
		return cached, nil
	}
	slugCacheMu.RUnlock()

	slugCacheMu.Lock()
	defer slugCacheMu.Unlock()

	if slugCache != nil {
		cached := make([]string, len(slugCache))
		copy(cached, slugCache)
		return cached, nil
	}

	slugs, err := s.queries.ListCategorySlugs(ctx)
	if err != nil {
		return nil, wrapErr("CategoryService.ListSlugs", err)
	}

	slugCache = slugs
	cached := make([]string, len(slugCache))
	copy(cached, slugCache)
	return cached, nil
}

func (s *catSvc) invalidateSlugCache() {
	slugCacheMu.Lock()
	slugCache = nil
	slugCacheMu.Unlock()
}

func (s *catSvc) ListWithUsage(ctx context.Context, userID uuid.UUID, startDate *string, endDate *string, limit *int32) ([]sqlc.ListCategoriesWithUsageRow, error) {
	var start, end *time.Time
	if startDate != nil {
		if parsed, err := time.Parse("2006-01-02", *startDate); err == nil {
			start = &parsed
		}
	}
	if endDate != nil {
		if parsed, err := time.Parse("2006-01-02", *endDate); err == nil {
			end = &parsed
		}
	}

	params := sqlc.ListCategoriesWithUsageParams{
		UserID:    userID,
		StartDate: start,
		EndDate:   end,
		Limit:     limit,
	}
	categories, err := s.queries.ListCategoriesWithUsage(ctx, params)
	if err != nil {
		return nil, wrapErr("CategoryService.ListWithUsage", err)
	}
	return categories, nil
}

func (s *catSvc) GetWithStats(ctx context.Context, userID uuid.UUID, id int64, startDate *string, endDate *string) (*sqlc.GetCategoryWithStatsRow, error) {
	var start, end *time.Time
	if startDate != nil {
		if parsed, err := time.Parse("2006-01-02", *startDate); err == nil {
			start = &parsed
		}
	}
	if endDate != nil {
		if parsed, err := time.Parse("2006-01-02", *endDate); err == nil {
			end = &parsed
		}
	}

	params := sqlc.GetCategoryWithStatsParams{
		UserID:    userID,
		ID:        id,
		StartDate: start,
		EndDate:   end,
	}
	stats, err := s.queries.GetCategoryWithStats(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("CategoryService.GetWithStats", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("CategoryService.GetWithStats", err)
	}
	return &stats, nil
}

func (s *catSvc) BulkCreate(ctx context.Context, categories []sqlc.BulkCreateCategoriesParams) error {
	_, err := s.queries.BulkCreateCategories(ctx, categories)
	if err != nil {
		return wrapErr("CategoryService.BulkCreate", err)
	}

	s.invalidateSlugCache()
	return nil
}

func (s *catSvc) DeleteUnused(ctx context.Context) error {
	_, err := s.queries.DeleteUnusedCategories(ctx)
	if err != nil {
		return wrapErr("CategoryService.DeleteUnused", err)
	}

	s.invalidateSlugCache()
	return nil
}

func (s *catSvc) Search(ctx context.Context, query string) ([]sqlc.Category, error) {
	categories, err := s.queries.SearchCategories(ctx, query)
	if err != nil {
		return nil, wrapErr("CategoryService.Search", err)
	}
	return categories, nil
}

func (s *catSvc) GetMostUsedForUser(ctx context.Context, params sqlc.GetMostUsedCategoriesForUserParams) ([]sqlc.GetMostUsedCategoriesForUserRow, error) {
	categories, err := s.queries.GetMostUsedCategoriesForUser(ctx, params)
	if err != nil {
		return nil, wrapErr("CategoryService.GetMostUsedForUser", err)
	}
	return categories, nil
}

func (s *catSvc) GetUnused(ctx context.Context) ([]sqlc.Category, error) {
	categories, err := s.queries.GetUnusedCategories(ctx)
	if err != nil {
		return nil, wrapErr("CategoryService.GetUnused", err)
	}
	return categories, nil
}

func (s *catSvc) GetCategoryUsageStats(ctx context.Context, userID uuid.UUID, id int64, startDate *string, endDate *string) (*sqlc.GetCategoryWithStatsRow, error) {
	return s.GetWithStats(ctx, userID, id, startDate, endDate)
}

func (s *catSvc) GetCategoriesWithStats(ctx context.Context, userID uuid.UUID, startDate *string, endDate *string, limit *int32) ([]sqlc.ListCategoriesWithUsageRow, error) {
	return s.ListWithUsage(ctx, userID, startDate, endDate, limit)
}
