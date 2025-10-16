package backup

import (
	"ariand/internal/db/sqlc"
	"ariand/internal/types"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func importCategories(ctx context.Context, db *sqlc.Queries, userID uuid.UUID, categories []CategoryData) error {
	for _, cat := range categories {
		if cat.Slug == "" || cat.Color == "" {
			return fmt.Errorf("category slug and color are required")
		}

		// Try to get existing category first
		_, err := db.GetCategoryBySlug(ctx, sqlc.GetCategoryBySlugParams{
			UserID: userID,
			Slug:   cat.Slug,
		})
		if err == nil {
			// Category exists, skip
			continue
		}

		// Category doesn't exist, create it
		_, err = db.CreateCategory(ctx, sqlc.CreateCategoryParams{
			UserID: userID,
			Slug:   cat.Slug,
			Color:  cat.Color,
		})
		if err != nil {
			return fmt.Errorf("failed to create category %q: %w", cat.Slug, err)
		}
	}

	return nil
}

func importAccounts(ctx context.Context, db *sqlc.Queries, userID uuid.UUID, accounts []AccountData) error {
	// Build existing accounts map to check for duplicates
	existingAccounts, err := buildAccountNameToIDMap(ctx, db, userID)
	if err != nil {
		return err
	}

	for _, acc := range accounts {
		if acc.Name == "" || acc.Bank == "" || acc.MainCurrency == "" {
			return fmt.Errorf("account name, bank, and main_currency are required")
		}

		// Skip if account already exists
		if _, exists := existingAccounts[acc.Name]; exists {
			continue
		}

		if acc.AnchorBalance == nil {
			return fmt.Errorf("anchor_balance is required for account %q", acc.Name)
		}

		accountType, err := parseAccountType(acc.AccountType)
		if err != nil {
			return fmt.Errorf("invalid account_type for %q: %w", acc.Name, err)
		}

		anchorBalanceBytes, err := types.ToBytes(acc.AnchorBalance)
		if err != nil {
			return fmt.Errorf("failed to convert anchor_balance for %q: %w", acc.Name, err)
		}

		colors := acc.Colors
		if colors == nil {
			colors = []string{}
		}

		_, err = db.CreateAccount(ctx, sqlc.CreateAccountParams{
			OwnerID:       userID,
			Name:          acc.Name,
			Bank:          acc.Bank,
			AccountType:   int16(accountType),
			Alias:         acc.Alias,
			AnchorBalance: anchorBalanceBytes,
			MainCurrency:  acc.MainCurrency,
			Colors:        colors,
		})
		if err != nil {
			return fmt.Errorf("failed to create account %q: %w", acc.Name, err)
		}
	}

	return nil
}

func importTransactions(ctx context.Context, db *sqlc.Queries, userID uuid.UUID, transactions []TransactionData) error {
	accountNameToID, err := buildAccountNameToIDMap(ctx, db, userID)
	if err != nil {
		return err
	}

	categorySlugToID, err := buildCategorySlugToIDMap(ctx, db, userID)
	if err != nil {
		return err
	}

	for _, tx := range transactions {
		if tx.AccountName == "" || tx.TxAmount == nil || tx.TxDirection == "" {
			return fmt.Errorf("transaction requires account_name, tx_amount, and tx_direction")
		}

		accountID := accountNameToID[tx.AccountName]
		if accountID == 0 {
			return fmt.Errorf("account %q not found", tx.AccountName)
		}

		txDirection, err := parseTransactionDirection(tx.TxDirection)
		if err != nil {
			return fmt.Errorf("invalid tx_direction: %w", err)
		}

		txAmountBytes, err := types.ToBytes(tx.TxAmount)
		if err != nil {
			return fmt.Errorf("failed to convert tx_amount: %w", err)
		}

		var balanceAfterBytes []byte
		if tx.BalanceAfter != nil {
			balanceAfterBytes, err = types.ToBytes(tx.BalanceAfter)
			if err != nil {
				return fmt.Errorf("failed to convert balance_after: %w", err)
			}
		}

		var foreignAmountBytes []byte
		if tx.ForeignAmount != nil {
			foreignAmountBytes, err = types.ToBytes(tx.ForeignAmount)
			if err != nil {
				return fmt.Errorf("failed to convert foreign_amount: %w", err)
			}
		}

		var categoryID *int64
		categoryManuallySet := false
		if tx.CategorySlug != nil && *tx.CategorySlug != "" {
			catID := categorySlugToID[*tx.CategorySlug]
			if catID == 0 {
				return fmt.Errorf("category %q not found", *tx.CategorySlug)
			}
			categoryID = &catID
			categoryManuallySet = true
		}

		merchantManuallySet := tx.Merchant != nil

		_, err = db.CreateTransaction(ctx, sqlc.CreateTransactionParams{
			UserID:              userID,
			AccountID:           accountID,
			TxDate:              tx.TxDate,
			TxAmount:            txAmountBytes,
			TxDirection:         int16(txDirection),
			TxDesc:              tx.TxDesc,
			BalanceAfter:        balanceAfterBytes,
			Merchant:            tx.Merchant,
			CategoryID:          categoryID,
			CategoryManuallySet: &categoryManuallySet,
			MerchantManuallySet: &merchantManuallySet,
			UserNotes:           tx.UserNotes,
			ForeignAmount:       foreignAmountBytes,
			ExchangeRate:        tx.ExchangeRate,
		})
		if err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}
	}

	return nil
}

func importRules(ctx context.Context, db *sqlc.Queries, userID uuid.UUID, rules []RuleData) error {
	categorySlugToID, err := buildCategorySlugToIDMap(ctx, db, userID)
	if err != nil {
		return err
	}

	// Get existing rules to check for duplicates
	existingRules, err := db.ListRules(ctx, userID)
	if err != nil {
		return err
	}

	existingRuleNames := make(map[string]bool)
	for _, r := range existingRules {
		existingRuleNames[r.RuleName] = true
	}

	for _, rule := range rules {
		if rule.RuleName == "" || rule.Conditions == nil {
			return fmt.Errorf("rule_name and conditions are required")
		}

		// Skip if rule already exists
		if existingRuleNames[rule.RuleName] {
			continue
		}

		var categoryID int64
		if rule.CategorySlug != nil && *rule.CategorySlug != "" {
			catID := categorySlugToID[*rule.CategorySlug]
			if catID == 0 {
				return fmt.Errorf("category %q not found", *rule.CategorySlug)
			}
			categoryID = catID
		}

		conditionsBytes, err := json.Marshal(rule.Conditions)
		if err != nil {
			return fmt.Errorf("failed to marshal conditions: %w", err)
		}

		merchant := ""
		if rule.Merchant != nil {
			merchant = *rule.Merchant
		}

		_, err = db.CreateRule(ctx, sqlc.CreateRuleParams{
			UserID:     userID,
			RuleName:   rule.RuleName,
			CategoryID: categoryID,
			Conditions: conditionsBytes,
			Merchant:   merchant,
		})
		if err != nil {
			return fmt.Errorf("failed to create rule %q: %w", rule.RuleName, err)
		}
	}

	return nil
}
