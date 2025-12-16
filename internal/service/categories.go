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
	Update(ctx context.Context, params sqlc.UpdateCategoryParams) error
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
	// hierarchical slugs like "food.groceries" require parent "food" to exist first
	if err := s.ensureParentCategories(ctx, params.UserID, params.Slug); err != nil {
		return nil, wrapErr("CategoryService.Create", err)
	}

	category, err := s.queries.CreateCategory(ctx, params)
	if err != nil {
		return nil, wrapErr("CategoryService.Create", err)
	}

	return &category, nil
}

func (s *catSvc) Update(ctx context.Context, params sqlc.UpdateCategoryParams) error {
	isSlugBeingUpdated := params.Slug != nil
	if isSlugBeingUpdated {
		oldCategory, err := s.queries.GetCategory(ctx, sqlc.GetCategoryParams{
			ID:     params.ID,
			UserID: params.UserID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return wrapErr("CategoryService.Update", ErrNotFound)
		}
		if err != nil {
			return wrapErr("CategoryService.Update", err)
		}

		slugIsActuallyChanging := oldCategory.Slug != *params.Slug
		if slugIsActuallyChanging {
			// when slug changes, we need to update child category slugs to maintain hierarchy
			if err := s.ensureParentCategories(ctx, params.UserID, *params.Slug); err != nil {
				return wrapErr("CategoryService.Update", err)
			}

			_, err = s.queries.UpdateChildCategorySlugs(ctx, sqlc.UpdateChildCategorySlugsParams{
				UserID:        params.UserID,
				OldSlugPrefix: oldCategory.Slug,
				NewSlugPrefix: *params.Slug,
			})
			if err != nil {
				return wrapErr("CategoryService.Update", err)
			}
		}
	}

	err := s.queries.UpdateCategory(ctx, params)
	if err != nil {
		return wrapErr("CategoryService.Update", err)
	}
	return nil
}

func (s *catSvc) Delete(ctx context.Context, userID uuid.UUID, id int64) (int64, error) {
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

	// deleting "food" should also delete "food.groceries", "food.dining", etc.
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

func (s *catSvc) ensureParentCategories(ctx context.Context, userID uuid.UUID, slug string) error {
	parts := strings.Split(slug, ".")
	hasNoParents := len(parts) <= 1
	if hasNoParents {
		return nil
	}

	// for "food.groceries.organic", create "food" then "food.groceries"
	for i := 1; i < len(parts); i++ {
		parentSlug := strings.Join(parts[:i], ".")

		parentExists, err := s.checkCategoryExists(ctx, userID, parentSlug)
		if err != nil {
			return err
		}
		if parentExists {
			continue
		}

		if err := s.createCategoryWithGeneratedColor(ctx, userID, parentSlug); err != nil {
			return err
		}
	}

	return nil
}

func (s *catSvc) checkCategoryExists(ctx context.Context, userID uuid.UUID, slug string) (bool, error) {
	_, err := s.queries.GetCategoryBySlug(ctx, sqlc.GetCategoryBySlugParams{
		Slug:   slug,
		UserID: userID,
	})

	if err == nil {
		return true, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return false, err
}

func (s *catSvc) createCategoryWithGeneratedColor(ctx context.Context, userID uuid.UUID, slug string) error {
	color := generateNiceHexColor()
	_, err := s.queries.CreateCategoryIfNotExists(ctx, sqlc.CreateCategoryIfNotExistsParams{
		UserID: userID,
		Slug:   slug,
		Color:  color,
	})
	return err
}

func generateNiceHexColor() string {
	// using chars 5-b creates muted colors
	niceHexChars := "56789ab"
	color := "#"

	randomBytes := make([]byte, 6)
	if _, err := rand.Read(randomBytes); err != nil {
		return "#888888"
	}

	for i := range 6 {
		charIndex := int(randomBytes[i]) % 7
		color += string(niceHexChars[charIndex])
	}

	return color
}
