package service

import (
	"ariand/internal/ai"
	"ariand/internal/db/sqlc"
	"ariand/internal/types"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

const (
	defaultAIProvider = "openai"
	defaultAIModel    = "gpt-4o-mini"
	maxDescQLength    = 100
)

type TransactionService interface {
	List(ctx context.Context, params sqlc.ListTransactionsParams) ([]sqlc.ListTransactionsRow, error)
	Get(ctx context.Context, params sqlc.GetTransactionParams) (*sqlc.GetTransactionRow, error)
	Create(ctx context.Context, params sqlc.CreateTransactionParams) (int64, error)
	Update(ctx context.Context, params sqlc.UpdateTransactionParams) error
	Delete(ctx context.Context, params sqlc.DeleteTransactionParams) (int64, error)
	BulkDelete(ctx context.Context, params sqlc.BulkDeleteTransactionsParams) error
	BulkCategorize(ctx context.Context, params sqlc.BulkCategorizeTransactionsParams) error
	GetTransactionCountByAccount(ctx context.Context, userID uuid.UUID) ([]sqlc.GetTransactionCountByAccountRow, error)
	FindCandidateTransactions(ctx context.Context, params sqlc.FindCandidateTransactionsParams) ([]sqlc.FindCandidateTransactionsRow, error)
	SetTransactionReceipt(ctx context.Context, params sqlc.SetTransactionReceiptParams) error
	CategorizeTransaction(ctx context.Context, userID uuid.UUID, txID int64) error
	IdentifyMerchantForTransaction(ctx context.Context, userID uuid.UUID, txID int64) error
	SearchTransactions(ctx context.Context, userID uuid.UUID, query string, accountID *int64, categoryID *int64, limit *int32, offset *int32) ([]sqlc.ListTransactionsRow, error)
	GetTransactionsByAccount(ctx context.Context, userID uuid.UUID, accountID int64, limit *int32, offset *int32) ([]sqlc.ListTransactionsRow, error)
	GetUncategorizedTransactions(ctx context.Context, userID uuid.UUID, accountID *int64, limit *int32, offset *int32) ([]sqlc.ListTransactionsRow, error)
}

type txnSvc struct {
	queries *sqlc.Queries
	log     *log.Logger
	catSvc  CategoryService
	aiMgr   *ai.Manager
}

func newTxnSvc(queries *sqlc.Queries, lg *log.Logger, catSvc CategoryService, aiMgr *ai.Manager) TransactionService {
	return &txnSvc{queries: queries, log: lg, catSvc: catSvc, aiMgr: aiMgr}
}

type categorizationResult struct {
	CategorySlug string
	Status       string
	Suggestions  []string
}

func (s *txnSvc) List(ctx context.Context, params sqlc.ListTransactionsParams) ([]sqlc.ListTransactionsRow, error) {
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

func (s *txnSvc) Get(ctx context.Context, params sqlc.GetTransactionParams) (*sqlc.GetTransactionRow, error) {
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

	return id, nil
}

func (s *txnSvc) Update(ctx context.Context, params sqlc.UpdateTransactionParams) error {
	_, err := s.queries.UpdateTransaction(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return wrapErr("TransactionService.Update", ErrNotFound)
	}

	if err != nil {
		return wrapErr("TransactionService.Update", err)
	}

	return nil
}

func (s *txnSvc) Delete(ctx context.Context, params sqlc.DeleteTransactionParams) (int64, error) {
	id, err := s.queries.DeleteTransaction(ctx, params)
	if err != nil {
		return 0, wrapErr("TransactionService.Delete", err)
	}
	return id, nil
}

func (s *txnSvc) BulkDelete(ctx context.Context, params sqlc.BulkDeleteTransactionsParams) error {
	_, err := s.queries.BulkDeleteTransactions(ctx, params)
	if err != nil {
		return wrapErr("TransactionService.BulkDelete", err)
	}
	return nil
}

func (s *txnSvc) BulkCategorize(ctx context.Context, params sqlc.BulkCategorizeTransactionsParams) error {
	_, err := s.queries.BulkCategorizeTransactions(ctx, params)
	if err != nil {
		return wrapErr("TransactionService.BulkCategorize", err)
	}
	return nil
}

func (s *txnSvc) GetTransactionCountByAccount(ctx context.Context, userID uuid.UUID) ([]sqlc.GetTransactionCountByAccountRow, error) {
	counts, err := s.queries.GetTransactionCountByAccount(ctx, userID)
	if err != nil {
		return nil, wrapErr("TransactionService.GetTransactionCountByAccount", err)
	}
	return counts, nil
}

func (s *txnSvc) FindCandidateTransactions(ctx context.Context, params sqlc.FindCandidateTransactionsParams) ([]sqlc.FindCandidateTransactionsRow, error) {
	candidates, err := s.queries.FindCandidateTransactions(ctx, params)
	if err != nil {
		return nil, wrapErr("TransactionService.FindCandidateTransactions", err)
	}
	return candidates, nil
}

func (s *txnSvc) SetTransactionReceipt(ctx context.Context, params sqlc.SetTransactionReceiptParams) error {
	_, err := s.queries.SetTransactionReceipt(ctx, params)
	if err != nil {
		return wrapErr("TransactionService.SetTransactionReceipt", err)
	}
	return nil
}

func (s *txnSvc) CategorizeTransaction(ctx context.Context, userID uuid.UUID, txID int64) error {
	s.log.Info("CategorizeTransaction", "user", userID, "tx", txID, "method", "similarity")

	tx, err := s.queries.GetTransaction(ctx, sqlc.GetTransactionParams{
		UserID: userID,
		ID:     txID,
	})

	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("transaction %d: %w", txID, ErrNotFound)
	}

	if err != nil {
		return wrapErr("CategorizeTransaction.GetTransaction", err)
	}

	// Convert GetTransactionRow to Transaction for determineCategory
	txForCategory := &sqlc.Transaction{
		ID:           tx.ID,
		AccountID:    tx.AccountID,
		EmailID:      tx.EmailID,
		TxDate:       tx.TxDate,
		TxAmount:     tx.TxAmount,
		TxDirection:  tx.TxDirection,
		TxDesc:       tx.TxDesc,
		BalanceAfter: tx.BalanceAfter,
		Merchant:     tx.Merchant,
		CategoryID:   tx.CategoryID,
		CatStatus:    tx.CatStatus,
		Suggestions:  tx.Suggestions,
		UserNotes:    tx.UserNotes,
		CreatedAt:    tx.CreatedAt,
		UpdatedAt:    tx.UpdatedAt,
	}

	result, err := s.determineCategory(ctx, userID, txForCategory)
	if err != nil {
		return wrapErr("CategorizeTransaction.DetermineCategory", err)
	}

	var categoryID *int64 // will be nil if no category found
	if result.CategorySlug != "" {
		category, err := s.catSvc.BySlug(ctx, userID, result.CategorySlug)
		if err != nil {
			return wrapErr("CategorizeTransaction.FindCategoryBySlug", err)
		}
		categoryID = &category.ID
	}

	// use atomic update - only succeeds if cat_status is still 0 (uncategorized)
	params := sqlc.CategorizeTransactionAtomicParams{
		ID:          txID,
		UserID:      userID,
		CategoryID:  categoryID,
		CatStatus:   2, // AI categorization status
		Suggestions: result.Suggestions,
	}

	updated, err := s.queries.CategorizeTransactionAtomic(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		// transaction was already categorized by another request - that's OK
		s.log.Info("Transaction already categorized", "tx", txID)
		return nil
	}
	if err != nil {
		return wrapErr("CategorizeTransaction.AtomicUpdate", err)
	}

	s.log.Info("Transaction categorized", "tx", updated.ID, "status", updated.CatStatus)
	return nil
}

func (s *txnSvc) IdentifyMerchantForTransaction(ctx context.Context, userID uuid.UUID, txID int64) error {
	s.log.Info("IdentifyMerchantForTransaction", "user", userID, "tx", txID)

	tx, err := s.queries.GetTransaction(ctx, sqlc.GetTransactionParams{
		UserID: userID,
		ID:     txID,
	})

	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("transaction %d: %w", txID, ErrNotFound)
	}

	if err != nil {
		return wrapErr("IdentifyMerchantForTransaction.GetTransaction", err)
	}

	if tx.TxDesc == nil || *tx.TxDesc == "" {
		return fmt.Errorf("transaction has no description to analyze: %w", ErrValidation)
	}

	if s.aiMgr == nil {
		return fmt.Errorf("AI manager not available: %w", ErrValidation)
	}

	// get a provider from the manager
	provider, err := s.aiMgr.GetProvider(defaultAIProvider, defaultAIModel)
	if err != nil {
		return wrapErr("IdentifyMerchantForTransaction.GetProvider", err)
	}

	merchant, err := provider.ExtractMerchant(ctx, *tx.TxDesc)
	if err != nil {
		return wrapErr("IdentifyMerchantForTransaction.ExtractMerchant", err)
	}

	if merchant == "" {
		return nil // no merchant identified
	}

	params := sqlc.UpdateTransactionParams{
		ID:       txID,
		UserID:   userID,
		Merchant: &merchant,
	}

	_, err = s.queries.UpdateTransaction(ctx, params)
	return wrapErr("IdentifyMerchantForTransaction.UpdateTransaction", err)
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
				if m.ID == tx.ID || m.CategoryID == nil || m.CategorySlug == nil || m.TxDesc == nil {
					continue
				}
				// require amounts to exist before comparing
				if tx.TxAmount == nil || m.TxAmount == nil {
					continue
				}
				if similarity(desc, strings.ToLower(*m.TxDesc)) >= 0.7 &&
					amountClose(tx.TxAmount, m.TxAmount, 0.20) {
					s.log.Info("found similar transaction for auto-categorization",
						"txID", tx.ID, "similarTxID", m.ID)
					return &categorizationResult{CategorySlug: *m.CategorySlug, Status: "auto"}, nil
				}
			}
		}
	}

	// 2. fallback to AI if available
	if s.aiMgr != nil {
		if provider, err := s.aiMgr.GetProvider(defaultAIProvider, defaultAIModel); err == nil {
			s.log.Info("falling back to AI for categorization", "txID", tx.ID)

			slugs, err := s.catSvc.ListSlugs(ctx, userID)
			if err != nil {
				return nil, wrapErr("determineCategory.ListSlugs", err)
			}
			categorySlug, _, suggestions, err := provider.CategorizeTransaction(ctx, *tx, slugs)
			if err != nil {
				return nil, wrapErr("determineCategory.CategorizeTransaction", err)
			}
			return &categorizationResult{
				CategorySlug: categorySlug,
				Suggestions:  suggestions,
				Status:       "ai",
			}, nil
		}
	}

	// 3. not found
	return &categorizationResult{CategorySlug: "", Status: "failed", Suggestions: []string{}}, nil
}

// validateCreateParams validates transaction creation parameters
func (s *txnSvc) validateCreateParams(params sqlc.CreateTransactionParams) error {
	if len(params.TxAmount) == 0 {
		return fmt.Errorf("tx_amount cannot be empty: %w", ErrValidation)
	}

	// Try to parse the JSONB amount to validate it's not zero
	var wrapper types.MoneyWrapper
	if err := wrapper.Scan(params.TxAmount); err == nil {
		if wrapper.Money != nil && wrapper.Money.Units == 0 && wrapper.Money.Nanos == 0 {
			return fmt.Errorf("tx_amount cannot be zero: %w", ErrValidation)
		}
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

func (s *txnSvc) SearchTransactions(ctx context.Context, userID uuid.UUID, query string, accountID *int64, categoryID *int64, limit *int32, offset *int32) ([]sqlc.ListTransactionsRow, error) {
	params := sqlc.ListTransactionsParams{
		UserID: userID,
		DescQ:  &query,
		Limit:  limit,
	}
	if accountID != nil {
		params.AccountIds = []int64{*accountID}
	}
	// TODO: categoryID parameter is currently ignored in this MVP implementation
	// future enhancement would support category filtering

	_ = offset // offset also not implemented in current query

	return s.List(ctx, params)
}

func (s *txnSvc) GetTransactionsByAccount(ctx context.Context, userID uuid.UUID, accountID int64, limit *int32, offset *int32) ([]sqlc.ListTransactionsRow, error) {
	params := sqlc.ListTransactionsParams{
		UserID:     userID,
		AccountIds: []int64{accountID},
		Limit:      limit,
	}
	// TODO: offset not implemented in current query for MVP
	_ = offset

	return s.List(ctx, params)
}

func (s *txnSvc) GetUncategorizedTransactions(ctx context.Context, userID uuid.UUID, accountID *int64, limit *int32, offset *int32) ([]sqlc.ListTransactionsRow, error) {
	params := sqlc.ListTransactionsParams{
		UserID:        userID,
		Uncategorized: boolPtr(true),
		Limit:         limit,
	}
	if accountID != nil {
		params.AccountIds = []int64{*accountID}
	}
	// TODO: offset not implemented in current query for MVP
	_ = offset

	return s.List(ctx, params)
}

func boolPtr(b bool) *bool {
	return &b
}

// amountClose reports whether a and b are within tolerance
// (e.g. 0.2 == 20%) of each other.
func amountClose(a, b *types.MoneyWrapper, tolerance float64) bool {
	if tolerance < 0 || a == nil || b == nil {
		return false
	}

	// Convert to float64 for comparison
	aFloat := float64(a.Money.Units) + float64(a.Money.Nanos)/1e9
	bFloat := float64(b.Money.Units) + float64(b.Money.Nanos)/1e9

	if aFloat == bFloat {
		return true
	}

	// Calculate tolerance
	maxMag := aFloat
	if bFloat > aFloat {
		maxMag = bFloat
	}
	if maxMag < 0 {
		maxMag = -maxMag
	}

	limit := maxMag * tolerance
	diff := aFloat - bFloat
	if diff < 0 {
		diff = -diff
	}

	return diff <= limit
}
