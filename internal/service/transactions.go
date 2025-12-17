package service

import (
	"ariand/internal/db/sqlc"
	"ariand/internal/exchange"
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
	Create(ctx context.Context, userID uuid.UUID, params []sqlc.CreateTransactionParams) ([]sqlc.Transaction, error)
	Update(ctx context.Context, params sqlc.UpdateTransactionParams) error
	Delete(ctx context.Context, params sqlc.DeleteTransactionParams) (int64, error)
	BulkDelete(ctx context.Context, params sqlc.BulkDeleteTransactionsParams) error
	Categorize(ctx context.Context, params sqlc.BulkCategorizeTransactionsParams) error
}

type txnSvc struct {
	queries        *sqlc.Queries
	log            *log.Logger
	catSvc         CategoryService
	ruleSvc        RuleService
	exchangeClient *exchange.Client
}

func newTxnSvc(queries *sqlc.Queries, lg *log.Logger, catSvc CategoryService, ruleSvc RuleService, exchangeClient *exchange.Client) TransactionService {
	return &txnSvc{queries: queries, log: lg, catSvc: catSvc, ruleSvc: ruleSvc, exchangeClient: exchangeClient}
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

func (s *txnSvc) processForeignCurrency(ctx context.Context, userID uuid.UUID, params *sqlc.CreateTransactionParams) (*sqlc.CreateTransactionParams, error) {
	account, err := s.queries.GetAccount(ctx, sqlc.GetAccountParams{
		UserID: userID,
		ID:     params.AccountID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}

	if params.TxCurrency == account.AnchorCurrency {
		params.ForeignAmountCents = nil
		params.ForeignCurrency = nil
		params.ExchangeRate = nil
		return params, nil
	}

	foreignAmountCents := params.TxAmountCents
	foreignCurrency := params.TxCurrency

	rate, err := s.exchangeClient.GetExchangeRate(foreignCurrency, account.AnchorCurrency, &params.TxDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate from %s to %s: %w", foreignCurrency, account.AnchorCurrency, err)
	}

	params.TxAmountCents = int64(float64(foreignAmountCents) * rate)
	params.TxCurrency = account.AnchorCurrency
	params.ForeignAmountCents = &foreignAmountCents
	params.ForeignCurrency = &foreignCurrency
	params.ExchangeRate = &rate

	return params, nil
}

func (s *txnSvc) Create(ctx context.Context, userID uuid.UUID, paramsList []sqlc.CreateTransactionParams) ([]sqlc.Transaction, error) {
	if len(paramsList) == 0 {
		return nil, fmt.Errorf("TransactionService.Create: no transactions provided")
	}

	// validate all transactions first
	for i, params := range paramsList {
		if err := s.validateCreateParams(params); err != nil {
			return nil, fmt.Errorf("TransactionService.Create: transaction %d invalid: %w", i, err)
		}
	}

	// process foreign currency conversions
	for i := range paramsList {
		converted, err := s.processForeignCurrency(ctx, userID, &paramsList[i])
		if err != nil {
			return nil, fmt.Errorf("TransactionService.Create: transaction %d currency conversion failed: %w", i, err)
		}
		paramsList[i] = *converted
	}

	transactions := make([]sqlc.Transaction, 0, len(paramsList))
	for _, params := range paramsList {
		tx, err := s.queries.CreateTransaction(ctx, params)
		if err != nil {
			return nil, wrapErr("TransactionService.Create.Insert", err)
		}

		transactions = append(transactions, tx)
	}

	// sync balances for all affected accounts
	affectedAccounts := make(map[int64]bool)
	for _, tx := range transactions {
		affectedAccounts[tx.AccountID] = true
	}

	for accountID := range affectedAccounts {
		if err := s.queries.SyncAccountBalances(ctx, accountID); err != nil {
			s.log.Warn("failed to sync account balances", "account_id", accountID, "error", err)
		}
	}

	// apply rules to transactions that need it
	for _, tx := range transactions {
		shouldApplyRules := !tx.CategoryManuallySet || !tx.MerchantManuallySet
		if shouldApplyRules {
			s.applyRulesToTransaction(ctx, userID, tx.ID)
		}
	}

	return transactions, nil
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

	// sync balances if amount, date, or direction changed
	balanceFieldsChanged := params.TxAmountCents != nil || params.TxDate != nil || params.TxDirection != nil
	if balanceFieldsChanged {
		if err := s.queries.SyncAccountBalances(ctx, tx.AccountID); err != nil {
			s.log.Warn("failed to sync account balances after updating transaction", "tx_id", params.ID, "account_id", tx.AccountID, "error", err)
		}
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
	// get transaction to find account_id before deletion
	tx, err := s.queries.GetTransaction(ctx, sqlc.GetTransactionParams{
		UserID: params.UserID,
		ID:     params.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, wrapErr("TransactionService.Delete", ErrNotFound)
	}
	if err != nil {
		return 0, wrapErr("TransactionService.Delete.GetOriginal", err)
	}

	affectedRows, err := s.queries.DeleteTransaction(ctx, params)
	if err != nil {
		return 0, wrapErr("TransactionService.Delete", err)
	}

	// sync balances after deletion
	if err := s.queries.SyncAccountBalances(ctx, tx.AccountID); err != nil {
		s.log.Warn("failed to sync account balances after deleting transaction", "tx_id", params.ID, "account_id", tx.AccountID, "error", err)
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

	// sync balances for all affected accounts
	for _, accountID := range affectedAccounts {
		if err := s.queries.SyncAccountBalances(ctx, accountID); err != nil {
			s.log.Warn("failed to sync account balances after bulk delete", "account_id", accountID, "error", err)
		}
	}

	s.log.Debug("Bulk deleted transactions and synced balances", "affected_accounts", len(affectedAccounts))

	return nil
}

func (s *txnSvc) Categorize(ctx context.Context, params sqlc.BulkCategorizeTransactionsParams) error {
	_, err := s.queries.BulkCategorizeTransactions(ctx, params)
	if err != nil {
		return wrapErr("TransactionService.Categorize", err)
	}
	return nil
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
	tx, err := s.queries.GetTransaction(ctx, sqlc.GetTransactionParams{
		UserID: userID,
		ID:     txID,
	})
	if err != nil {
		s.log.Warn("failed to fetch transaction for rule application", "tx_id", txID, "error", err)
		return
	}

	bothFieldsManuallySet := tx.CategoryManuallySet && tx.MerchantManuallySet
	if bothFieldsManuallySet {
		return
	}

	account, err := s.queries.GetAccount(ctx, sqlc.GetAccountParams{
		UserID: userID,
		ID:     tx.AccountID,
	})
	if err != nil {
		s.log.Warn("failed to fetch account for rule application", "account_id", tx.AccountID, "error", err)
		return
	}

	result, err := s.ruleSvc.ApplyToTransaction(ctx, userID, &tx, &account)
	if err != nil {
		s.log.Warn("failed to apply rules", "tx_id", txID, "error", err)
		return
	}

	noRulesMatched := result.CategoryID == nil && result.Merchant == nil
	if noRulesMatched {
		return
	}

	updateParams := sqlc.UpdateTransactionParams{
		ID:     txID,
		UserID: userID,
	}

	if !tx.CategoryManuallySet && result.CategoryID != nil {
		updateParams.CategoryID = result.CategoryID
	}

	if !tx.MerchantManuallySet && result.Merchant != nil {
		updateParams.Merchant = result.Merchant
	}

	if err := s.queries.UpdateTransaction(ctx, updateParams); err != nil {
		s.log.Warn("failed to update transaction with rule results", "tx_id", txID, "error", err)
	}
}
