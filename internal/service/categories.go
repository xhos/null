package service

import (
	"ariand/internal/db/sqlc"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"

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
	// Create parent categories if they don't exist
	if err := s.ensureParentCategories(ctx, params.UserID, params.Slug); err != nil {
		return nil, wrapErr("CategoryService.Create", err)
	}

	category, err := s.queries.CreateCategory(ctx, params)
	if err != nil {
		return nil, wrapErr("CategoryService.Create", err)
	}

	return &category, nil
}

func (s *catSvc) Update(ctx context.Context, params sqlc.UpdateCategoryParams) (*sqlc.Category, error) {
	category, err := s.queries.UpdateCategory(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("CategoryService.Update", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("CategoryService.Update", err)
	}

	return &category, nil
}

func (s *catSvc) Delete(ctx context.Context, userID uuid.UUID, id int64) (int64, error) {
	// First get the category to find its slug
	category, err := s.queries.GetCategory(ctx, sqlc.GetCategoryParams{
		ID:     id,
		UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, wrapErr("CategoryService.Delete", ErrNotFound)
	}
	if err != nil {
		return 0, wrapErr("CategoryService.Delete", err)
	}

	// Delete this category and all children (cascading delete)
	affected, err := s.queries.DeleteCategoriesBySlugPrefix(ctx, sqlc.DeleteCategoriesBySlugPrefixParams{
		UserID: userID,
		Slug:   category.Slug,
	})
	if err != nil {
		return 0, wrapErr("CategoryService.Delete", err)
	}

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

// ensureParentCategories creates all parent categories if they don't exist
func (s *catSvc) ensureParentCategories(ctx context.Context, userID uuid.UUID, slug string) error {
	parts := strings.Split(slug, ".")
	if len(parts) <= 1 {
		return nil // No parents needed
	}

	// Create each parent category if it doesn't exist
	for i := 1; i < len(parts); i++ {
		parentSlug := strings.Join(parts[:i], ".")

		// Check if parent exists
		_, err := s.queries.GetCategoryBySlug(ctx, sqlc.GetCategoryBySlugParams{
			Slug:   parentSlug,
			UserID: userID,
		})

		if err == nil {
			continue // Parent exists
		}

		if !errors.Is(err, sql.ErrNoRows) {
			return err // Other error
		}

		// Parent doesn't exist, create it with a generated color
		color := generateNiceHexColor()
		_, err = s.queries.CreateCategoryIfNotExists(ctx, sqlc.CreateCategoryIfNotExistsParams{
			UserID: userID,
			Slug:   parentSlug,
			Color:  color,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// generateNiceHexColor generates a nice muted hex color
func generateNiceHexColor() string {
	niceHexChars := "56789ab"
	color := "#"

	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		// Fallback if crypto/rand fails
		return "#888888"
	}

	for i := 0; i < 6; i++ {
		color += string(niceHexChars[int(b[i])%7])
	}

	return color
}
