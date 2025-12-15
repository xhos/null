package service

import (
	"ariand/internal/db/sqlc"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

const (
	maxDescQLength = 100
)

type TransactionService interface {
	List(ctx context.Context, params sqlc.ListTransactionsParams) ([]sqlc.Transaction, error)
	Get(ctx context.Context, params sqlc.GetTransactionParams) (*sqlc.Transaction, error)
	Create(ctx context.Context, params sqlc.CreateTransactionParams) (int64, error)
	Update(ctx context.Context, params sqlc.UpdateTransactionParams) error
	Delete(ctx context.Context, params sqlc.DeleteTransactionParams) (int64, error)
	BulkDelete(ctx context.Context, params sqlc.BulkDeleteTransactionsParams) error
	Categorize(ctx context.Context, params sqlc.BulkCategorizeTransactionsParams) error
}

type txnSvc struct {
	queries *sqlc.Queries
	log     *log.Logger
	catSvc  CategoryService
	ruleSvc RuleService
}

func newTxnSvc(queries *sqlc.Queries, lg *log.Logger, catSvc CategoryService, ruleSvc RuleService) TransactionService {
	return &txnSvc{queries: queries, log: lg, catSvc: catSvc, ruleSvc: ruleSvc}
}

type categorizationResult struct {
	CategorySlug string
	Status       string
	Suggestions  []string
}

func (s *txnSvc) List(ctx context.Context, params sqlc.ListTransactionsParams) ([]sqlc.Transaction, error) {
	// truncate overly long description queries for performance
	if params.DescQ != nil && len(*params.DescQ) > maxDescQLength {
		truncated := (*params.DescQ)[:maxDescQLength]
		params.DescQ = &truncated
	}

	rows, err := s.queries.ListTransactions(ctx, params)
	if err != nil {
		return nil, wrapErr("TransactionService.List", err)
	}

	return rows, nil
}

func (s *txnSvc) Get(ctx context.Context, params sqlc.GetTransactionParams) (*sqlc.Transaction, error) {
	row, err := s.queries.GetTransaction(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("TransactionService.Get", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("TransactionService.Get", err)
	}

	return &row, nil
}

func (s *txnSvc) Create(ctx context.Context, params sqlc.CreateTransactionParams) (int64, error) {
	if err := s.validateCreateParams(params); err != nil {
		return 0, fmt.Errorf("TransactionService.Create: %w", err)
	}

	id, err := s.queries.CreateTransaction(ctx, params)
	if err != nil {
		return 0, wrapErr("TransactionService.Create", err)
	}

	// apply rules if fields weren't manually set
	shouldApplyRules := (params.CategoryManuallySet == nil || !*params.CategoryManuallySet) ||
		(params.MerchantManuallySet == nil || !*params.MerchantManuallySet)

	s.log.Info("Transaction created, checking if rules should apply",
		"tx_id", id,
		"should_apply_rules", shouldApplyRules,
		"category_manually_set", params.CategoryManuallySet,
		"merchant_manually_set", params.MerchantManuallySet)

	if shouldApplyRules {
		s.applyRulesToTransaction(ctx, params.UserID, id)
	}

	return id, nil
}

func (s *txnSvc) Update(ctx context.Context, params sqlc.UpdateTransactionParams) error {
	// get current transaction to check what changed
	tx, err := s.queries.GetTransaction(ctx, sqlc.GetTransactionParams{
		UserID: params.UserID,
		ID:     params.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return wrapErr("TransactionService.Update", ErrNotFound)
	}
	if err != nil {
		return wrapErr("TransactionService.Update.GetOriginal", err)
	}

	err = s.queries.UpdateTransaction(ctx, params)
	if err != nil {
		return wrapErr("TransactionService.Update", err)
	}

	// apply rules if relevant fields changed and aren't manually set
	fieldsChangedForRules := params.TxDesc != nil || params.Merchant != nil || params.TxAmountCents != nil
	shouldApplyRules := fieldsChangedForRules &&
		(!tx.CategoryManuallySet || !tx.MerchantManuallySet)

	if shouldApplyRules {
		s.applyRulesToTransaction(ctx, params.UserID, params.ID)
	}

	return nil
}

func (s *txnSvc) Delete(ctx context.Context, params sqlc.DeleteTransactionParams) (int64, error) {
	affectedRows, err := s.queries.DeleteTransaction(ctx, params)
	if err != nil {
		return 0, wrapErr("TransactionService.Delete", err)
	}

	return affectedRows, nil
}

func (s *txnSvc) BulkDelete(ctx context.Context, params sqlc.BulkDeleteTransactionsParams) error {
	// Get list of affected accounts before deletion
	affectedAccounts, err := s.queries.GetAccountIDsFromTransactionIDs(ctx, params.TransactionIds)
	if err != nil {
		return wrapErr("TransactionService.BulkDelete.GetAccounts", err)
	}

	_, err = s.queries.BulkDeleteTransactions(ctx, params)
	if err != nil {
		return wrapErr("TransactionService.BulkDelete", err)
	}

	s.log.Debug("Bulk deleted transactions, balances will be recalculated by DB", "affected_accounts", len(affectedAccounts))

	return nil
}

func (s *txnSvc) Categorize(ctx context.Context, params sqlc.BulkCategorizeTransactionsParams) error {
	_, err := s.queries.BulkCategorizeTransactions(ctx, params)
	if err != nil {
		return wrapErr("TransactionService.Categorize", err)
	}
	return nil
}

// determineCategory analyzes a transaction to suggest a category
func (s *txnSvc) determineCategory(ctx context.Context, userID uuid.UUID, tx *sqlc.Transaction) (*categorizationResult, error) {
	// 1. rule-based similarity (fast path)
	if tx.TxDesc != nil {
		params := sqlc.ListTransactionsParams{
			UserID: userID,
			DescQ:  tx.TxDesc,
			Limit:  int32Ptr(10),
		}
		if rows, err := s.queries.ListTransactions(ctx, params); err == nil {
			desc := strings.ToLower(*tx.TxDesc)
			for _, m := range rows {
				// must be a different txn with usable fields
				if m.ID == tx.ID || m.CategoryID == nil || m.TxDesc == nil {
					continue
				}

				// Check if amounts are within 20% of each other
				amountDiff := tx.TxAmountCents - m.TxAmountCents
				if amountDiff < 0 {
					amountDiff = -amountDiff
				}
				maxAmount := tx.TxAmountCents
				if m.TxAmountCents > maxAmount {
					maxAmount = m.TxAmountCents
				}
				withinTolerance := maxAmount == 0 || float64(amountDiff) <= float64(maxAmount)*0.20

				if similarity(desc, strings.ToLower(*m.TxDesc)) >= 0.7 && withinTolerance {
					// fetch category to get slug
					category, err := s.catSvc.Get(ctx, userID, *m.CategoryID)
					if err != nil {
						continue
					}
					s.log.Info("found similar transaction for auto-categorization",
						"txID", tx.ID, "similarTxID", m.ID)
					return &categorizationResult{CategorySlug: category.Slug, Status: "auto"}, nil
				}
			}
		}
	}

	// not found
	return &categorizationResult{CategorySlug: "", Status: "failed", Suggestions: []string{}}, nil
}

// validateCreateParams validates transaction creation parameters
func (s *txnSvc) validateCreateParams(params sqlc.CreateTransactionParams) error {
	if params.TxAmountCents == 0 {
		return fmt.Errorf("tx_amount cannot be zero: %w", ErrValidation)
	}

	switch params.TxDirection {
	case 1, 2: // DIRECTION_INCOMING, DIRECTION_OUTGOING
		// valid
	default:
		return fmt.Errorf("tx_direction must be 1 (DIRECTION_INCOMING) or 2 (DIRECTION_OUTGOING): %w", ErrValidation)
	}

	return nil
}

func int32Ptr(i int32) *int32 {
	return &i
}

func similarity(a, b string) float64 {
	aa := strings.Fields(a)
	bb := strings.Fields(b)
	set := map[string]bool{}
	for _, w := range aa {
		set[w] = true
	}

	inter := 0
	for _, w := range bb {
		if set[w] {
			inter++
		}
	}

	union := len(aa) + len(bb) - inter
	if union == 0 {
		return 0
	}

	return float64(inter) / float64(union)
}

func boolPtr(b bool) *bool {
	return &b
}

func (s *txnSvc) applyRulesToTransaction(ctx context.Context, userID uuid.UUID, txID int64) {
	s.log.Info("Applying rules to transaction", "tx_id", txID, "user_id", userID)

	// fetch transaction with account data for rule evaluation
	tx, err := s.queries.GetTransaction(ctx, sqlc.GetTransactionParams{
		UserID: userID,
		ID:     txID,
	})
	if err != nil {
		s.log.Warn("failed to fetch transaction for rule application", "tx_id", txID, "error", err)
		return
	}

	// skip if manually set
	if tx.CategoryManuallySet && tx.MerchantManuallySet {
		s.log.Info("Skipping rule application - both fields manually set", "tx_id", txID)
		return
	}

	// fetch account for rule evaluation
	account, err := s.queries.GetAccount(ctx, sqlc.GetAccountParams{
		UserID: userID,
		ID:     tx.AccountID,
	})
	if err != nil {
		s.log.Warn("failed to fetch account for rule application", "account_id", tx.AccountID, "error", err)
		return
	}

	s.log.Info("Transaction data for rule evaluation",
		"tx_id", txID,
		"description", tx.TxDesc,
		"merchant", tx.Merchant,
		"category_manually_set", tx.CategoryManuallySet,
		"merchant_manually_set", tx.MerchantManuallySet)

	// apply rules
	result, err := s.ruleSvc.ApplyToTransaction(ctx, userID, &tx, &account)
	if err != nil {
		s.log.Warn("failed to apply rules", "tx_id", txID, "error", err)
		return
	}

	s.log.Info("Rule application result",
		"tx_id", txID,
		"category_id", result.CategoryID,
		"merchant", result.Merchant)

	// update only if rules matched something
	if result.CategoryID == nil && result.Merchant == nil {
		s.log.Info("No rules matched for transaction", "tx_id", txID)
		return
	}

	updateParams := sqlc.UpdateTransactionParams{
		ID:     txID,
		UserID: userID,
	}

	if !tx.CategoryManuallySet && result.CategoryID != nil {
		updateParams.CategoryID = result.CategoryID
		s.log.Info("Setting category from rule", "tx_id", txID, "category_id", *result.CategoryID)
	}

	if !tx.MerchantManuallySet && result.Merchant != nil {
		updateParams.Merchant = result.Merchant
		s.log.Info("Setting merchant from rule", "tx_id", txID, "merchant", *result.Merchant)
	}

	err = s.queries.UpdateTransaction(ctx, updateParams)
	if err != nil {
		s.log.Warn("failed to update transaction with rule results", "tx_id", txID, "error", err)
		return
	}

	s.log.Info("Successfully applied rules to transaction", "tx_id", txID)
}
