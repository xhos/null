package service

import (
	"ariand/internal/db/sqlc"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

type CategoryService interface {
	List(ctx context.Context, userID uuid.UUID) ([]sqlc.Category, error)
	Get(ctx context.Context, userID uuid.UUID, id int64) (*sqlc.Category, error)
	Create(ctx context.Context, params sqlc.CreateCategoryParams) (*sqlc.Category, error)
	Update(ctx context.Context, params sqlc.UpdateCategoryParams) (*sqlc.Category, error)
	Delete(ctx context.Context, userID uuid.UUID, id int64) (int64, error)
	BySlug(ctx context.Context, userID uuid.UUID, slug string) (*sqlc.Category, error)
	ListSlugs(ctx context.Context, userID uuid.UUID) ([]string, error)
}

var (
	slugCache   map[uuid.UUID][]string
	slugCacheMu sync.RWMutex
	slugRegex   = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9]+)*$`)
)

func init() {
	slugCache = make(map[uuid.UUID][]string)
}

type catSvc struct {
	queries *sqlc.Queries
	log     *log.Logger
}

func newCatSvc(queries *sqlc.Queries, lg *log.Logger) CategoryService {
	return &catSvc{queries: queries, log: lg}
}

func (s *catSvc) List(ctx context.Context, userID uuid.UUID) ([]sqlc.Category, error) {
	categories, err := s.queries.ListCategories(ctx, userID)
	if err != nil {
		return nil, wrapErr("CategoryService.List", err)
	}
	return categories, nil
}

func (s *catSvc) Get(ctx context.Context, userID uuid.UUID, id int64) (*sqlc.Category, error) {
	category, err := s.queries.GetCategory(ctx, sqlc.GetCategoryParams{
		ID:     id,
		UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("CategoryService.Get", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("CategoryService.Get", err)
	}
	return &category, nil
}

func (s *catSvc) Create(ctx context.Context, params sqlc.CreateCategoryParams) (*sqlc.Category, error) {
	isValidSlug := slugRegex.MatchString(params.Slug)
	if !isValidSlug {
		return nil, wrapErr("CategoryService.Create", ErrValidation)
	}

	category, err := s.queries.CreateCategory(ctx, params)
	if err != nil {
		return nil, wrapErr("CategoryService.Create", err)
	}

	s.invalidateSlugCache(params.UserID)

	return &category, nil
}

func (s *catSvc) Update(ctx context.Context, params sqlc.UpdateCategoryParams) (*sqlc.Category, error) {
	if params.Slug != nil {
		isValidSlug := slugRegex.MatchString(*params.Slug)
		if !isValidSlug {
			return nil, wrapErr("CategoryService.Update", ErrValidation)
		}
	}

	category, err := s.queries.UpdateCategory(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("CategoryService.Update", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("CategoryService.Update", err)
	}

	s.invalidateSlugCache(params.UserID)

	return &category, nil
}

func (s *catSvc) Delete(ctx context.Context, userID uuid.UUID, id int64) (int64, error) {
	affected, err := s.queries.DeleteCategory(ctx, sqlc.DeleteCategoryParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return 0, wrapErr("CategoryService.Delete", err)
	}

	s.invalidateSlugCache(userID)

	return affected, nil
}

func (s *catSvc) BySlug(ctx context.Context, userID uuid.UUID, slug string) (*sqlc.Category, error) {
	category, err := s.queries.GetCategoryBySlug(ctx, sqlc.GetCategoryBySlugParams{
		Slug:   slug,
		UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("CategoryService.BySlug", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("CategoryService.BySlug", err)
	}
	return &category, nil
}

func (s *catSvc) ListSlugs(ctx context.Context, userID uuid.UUID) ([]string, error) {
	slugCacheMu.RLock()
	cached, exists := slugCache[userID]
	if exists {
		result := make([]string, len(cached))
		copy(result, cached)
		slugCacheMu.RUnlock()
		return result, nil
	}
	slugCacheMu.RUnlock()

	slugCacheMu.Lock()
	defer slugCacheMu.Unlock()

	// Double-check after acquiring write lock
	cached, exists = slugCache[userID]
	if exists {
		result := make([]string, len(cached))
		copy(result, cached)
		return result, nil
	}

	slugs, err := s.queries.ListCategorySlugs(ctx, userID)
	if err != nil {
		return nil, wrapErr("CategoryService.ListSlugs", err)
	}

	slugCache[userID] = slugs
	result := make([]string, len(slugs))
	copy(result, slugs)
	return result, nil
}

func (s *catSvc) invalidateSlugCache(userID uuid.UUID) {
	slugCacheMu.Lock()
	delete(slugCache, userID)
	slugCacheMu.Unlock()
}
