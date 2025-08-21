package service

import (
	"ariand/internal/db/sqlc"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"sync"

	"github.com/charmbracelet/log"
)

type CategoryService interface {
	List(ctx context.Context) ([]sqlc.Category, error)
	Get(ctx context.Context, id int64) (*sqlc.Category, error)
	Create(ctx context.Context, params sqlc.CreateCategoryParams) (*sqlc.Category, error)
	Update(ctx context.Context, params sqlc.UpdateCategoryParams) (*sqlc.Category, error)
	Delete(ctx context.Context, id int64) (int64, error)
	BySlug(ctx context.Context, slug string) (*sqlc.Category, error)
	ListSlugs(ctx context.Context) ([]string, error)
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

func (s *catSvc) Delete(ctx context.Context, id int64) (int64, error) {
	affected, err := s.queries.DeleteCategory(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, wrapErr("CategoryService.Delete", ErrNotFound)
	}
	if err != nil {
		return 0, wrapErr("CategoryService.Delete", err)
	}

	s.invalidateSlugCache()

	return affected, nil
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
