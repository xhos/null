package backup

import (
	"ariand/internal/db/sqlc"
	arian "ariand/internal/gen/arian/v1"
	"ariand/internal/types"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func exportCategories(ctx context.Context, db *sqlc.Queries, userID uuid.UUID) ([]CategoryData, error) {
	categories, err := db.ListCategories(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]CategoryData, len(categories))
	for i, cat := range categories {
		result[i] = CategoryData{
			Slug:  cat.Slug,
			Color: cat.Color,
		}
	}

	return result, nil
}

func exportAccounts(ctx context.Context, db *sqlc.Queries, userID uuid.UUID) ([]AccountData, error) {
	accounts, err := db.ListAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]AccountData, len(accounts))
	for i, acc := range accounts {
		data := AccountData{
			Name:         acc.Name,
			Bank:         acc.Bank,
			AccountType:  formatAccountType(arian.AccountType(acc.AccountType)),
			Alias:        acc.Alias,
			MainCurrency: acc.MainCurrency,
			Colors:       acc.Colors,
		}

		if !acc.AnchorDate.IsZero() {
			data.AnchorDate = &acc.AnchorDate
		}

		if acc.AnchorBalance != nil {
			data.AnchorBalance = types.Unwrap(acc.AnchorBalance)
		}

		result[i] = data
	}

	return result, nil
}

func exportTransactions(ctx context.Context, db *sqlc.Queries, userID uuid.UUID) ([]TransactionData, error) {
	transactions, err := db.ListAllTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}

	accountMap, err := buildAccountMap(ctx, db, userID)
	if err != nil {
		return nil, err
	}

	categoryMap, err := buildCategoryMap(ctx, db, userID)
	if err != nil {
		return nil, err
	}

	result := make([]TransactionData, len(transactions))
	for i, tx := range transactions {
		accountName := accountMap[tx.AccountID]
		if accountName == "" {
			return nil, fmt.Errorf("account ID %d not found", tx.AccountID)
		}

		data := TransactionData{
			AccountName:  accountName,
			TxDate:       tx.TxDate,
			TxAmount:     types.Unwrap(tx.TxAmount),
			TxDirection:  formatTransactionDirection(arian.TransactionDirection(tx.TxDirection)),
			TxDesc:       tx.TxDesc,
			Merchant:     tx.Merchant,
			UserNotes:    tx.UserNotes,
			ExchangeRate: tx.ExchangeRate,
		}

		if tx.BalanceAfter != nil {
			data.BalanceAfter = types.Unwrap(tx.BalanceAfter)
		}

		if tx.ForeignAmount != nil {
			data.ForeignAmount = types.Unwrap(tx.ForeignAmount)
		}

		if tx.CategoryID != nil {
			slug := categoryMap[*tx.CategoryID]
			data.CategorySlug = &slug
		}

		result[i] = data
	}

	return result, nil
}

func exportRules(ctx context.Context, db *sqlc.Queries, userID uuid.UUID) ([]RuleData, error) {
	rules, err := db.ListRules(ctx, userID)
	if err != nil {
		return nil, err
	}

	categoryMap, err := buildCategoryMap(ctx, db, userID)
	if err != nil {
		return nil, err
	}

	result := make([]RuleData, len(rules))
	for i, rule := range rules {
		var conditions map[string]interface{}
		if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal conditions for rule %q: %w", rule.RuleName, err)
		}

		data := RuleData{
			RuleName:      rule.RuleName,
			Merchant:      rule.Merchant,
			Conditions:    conditions,
			IsActive:      rule.IsActive,
			PriorityOrder: &rule.PriorityOrder,
			RuleSource:    &rule.RuleSource,
		}

		if rule.CategoryID != nil {
			slug := categoryMap[*rule.CategoryID]
			data.CategorySlug = &slug
		}

		result[i] = data
	}

	return result, nil
}
