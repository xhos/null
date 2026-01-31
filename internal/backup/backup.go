package backup

import (
	"context"
	"fmt"
	"time"

	"null/internal/db/sqlc"

	"github.com/google/uuid"
)

func ExportAll(ctx context.Context, db *sqlc.Queries, userID uuid.UUID) (*Backup, error) {
	backup := &Backup{
		Version:    "1.0",
		ExportedAt: time.Now(),
	}

	categories, err := exportCategories(ctx, db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to export categories: %w", err)
	}
	backup.Categories = categories

	accounts, err := exportAccounts(ctx, db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to export accounts: %w", err)
	}
	backup.Accounts = accounts

	transactions, err := exportTransactions(ctx, db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to export transactions: %w", err)
	}
	backup.Transactions = transactions

	rules, err := exportRules(ctx, db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to export rules: %w", err)
	}
	backup.Rules = rules

	return backup, nil
}

func ImportAll(ctx context.Context, db *sqlc.Queries, userID uuid.UUID, backup *Backup) error {
	if err := importCategories(ctx, db, userID, backup.Categories); err != nil {
		return fmt.Errorf("failed to import categories: %w", err)
	}

	if err := importAccounts(ctx, db, userID, backup.Accounts); err != nil {
		return fmt.Errorf("failed to import accounts: %w", err)
	}

	if err := importTransactions(ctx, db, userID, backup.Transactions); err != nil {
		return fmt.Errorf("failed to import transactions: %w", err)
	}

	if err := importRules(ctx, db, userID, backup.Rules); err != nil {
		return fmt.Errorf("failed to import rules: %w", err)
	}

	return nil
}
