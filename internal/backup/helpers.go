package backup

import (
	"context"

	"null-core/internal/db/sqlc"

	"github.com/google/uuid"
)

func buildAccountMap(ctx context.Context, db *sqlc.Queries, userID uuid.UUID) (map[int64]string, error) {
	accounts, err := db.ListAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}

	accountMap := make(map[int64]string)
	for _, acc := range accounts {
		accountMap[acc.Account.ID] = acc.Account.Name
	}

	return accountMap, nil
}

func buildCategoryMap(ctx context.Context, db *sqlc.Queries, userID uuid.UUID) (map[int64]string, error) {
	categories, err := db.ListCategories(ctx, userID)
	if err != nil {
		return nil, err
	}

	categoryMap := make(map[int64]string)
	for _, cat := range categories {
		categoryMap[cat.ID] = cat.Slug
	}

	return categoryMap, nil
}

func buildAccountNameToIDMap(ctx context.Context, db *sqlc.Queries, userID uuid.UUID) (map[string]int64, error) {
	accounts, err := db.ListAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}

	accountMap := make(map[string]int64)
	for _, acc := range accounts {
		accountMap[acc.Account.Name] = acc.Account.ID
	}

	return accountMap, nil
}

func buildCategorySlugToIDMap(ctx context.Context, db *sqlc.Queries, userID uuid.UUID) (map[string]int64, error) {
	categories, err := db.ListCategories(ctx, userID)
	if err != nil {
		return nil, err
	}

	categoryMap := make(map[string]int64)
	for _, cat := range categories {
		categoryMap[cat.Slug] = cat.ID
	}

	return categoryMap, nil
}
