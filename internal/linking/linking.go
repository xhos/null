package linking

// TODO: Temporarily commented out during MoneyWrapper migration
// This needs to be updated to work with the new JSONB money types

/*
import (
	"ariand/internal/db/sqlc"
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)
*/

/*
type Service interface {
	LinkReceiptToTransaction(ctx context.Context, receiptID int64) error
}

type service struct {
	queries *sqlc.Queries
}

func NewService(queries *sqlc.Queries) Service {
	return &service{queries: queries}
}

type transactionMatch struct {
	transaction *sqlc.FindCandidateTransactionsForUserRow
	score       float64
}

func (s *service) LinkReceiptToTransaction(ctx context.Context, receiptID int64) error {
	receipt, err := s.queries.GetReceipt(ctx, receiptID)
	if err != nil {
		return err
	}

	// find candidate transactions if we have amount
	if receipt.TotalAmount == nil {
		// can't match without amount
		return s.setLinkStatus(ctx, receiptID, 1) // unlinked
	}

	candidates, err := s.findCandidateTransactions(ctx, receipt)
	if err != nil {
		return err
	}

	if len(candidates) == 0 {
		return s.setLinkStatus(ctx, receiptID, 1) // unlinked
	}

	matches := s.scoreMatches(candidates, receipt)
	if len(matches) == 0 {
		return s.setLinkStatus(ctx, receiptID, 1) // unlinked
	}

	best := matches[0]
	linkStatus := int16(3) // needs verification
	if best.score >= 0.95 {
		linkStatus = 2 // matched
	}

	// collect match ids for suggestions
	var matchIDs []int64
	for _, match := range matches {
		matchIDs = append(matchIDs, match.transaction.ID)
	}

	return s.updateReceiptWithMatch(ctx, receiptID, best.transaction.ID, linkStatus, matchIDs)
}

func (s *service) findCandidateTransactions(ctx context.Context, receipt sqlc.Receipt) ([]sqlc.FindCandidateTransactionsForUserRow, error) {
	if receipt.TotalAmount == nil {
		return nil, nil
	}

	receiptAmount := *receipt.TotalAmount

	// need user ID from context - this is a breaking change in the SQLC interface
	merchant := ""
	if receipt.Merchant != nil {
		merchant = *receipt.Merchant
	}

	// Use purchase date directly as time.Time
	var targetDate time.Time
	if receipt.PurchaseDate != nil {
		targetDate = *receipt.PurchaseDate
	}

	// TODO: get userID from calling context - this function signature changed
	// For now, use a placeholder UUID to fix compilation
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000000")

	return s.queries.FindCandidateTransactionsForUser(ctx, sqlc.FindCandidateTransactionsForUserParams{
		Merchant: merchant,
		UserID:   userID,
		Date:     targetDate,
		Total:    receiptAmount,
	})
}

func (s *service) scoreMatches(candidates []sqlc.FindCandidateTransactionsForUserRow, receipt sqlc.Receipt) []transactionMatch {
	var matches []transactionMatch

	for _, tx := range candidates {
		score := s.calculateScore(tx, receipt)
		if score >= 0.7 { // threshold
			matches = append(matches, transactionMatch{
				transaction: &tx,
				score:       score,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	return matches
}

func (s *service) calculateScore(tx sqlc.FindCandidateTransactionsForUserRow, receipt sqlc.Receipt) float64 {
	amountScore := s.scoreAmount(tx.TxAmount, receipt.TotalAmount)
	dateScore := s.scoreDateMatch(tx.TxDate, receipt.PurchaseDate)
	merchantScore := s.scoreMerchant(tx.TxDesc, tx.Merchant, receipt.Merchant)

	return amountScore*0.45 + dateScore*0.35 + merchantScore*0.2
}

func (s *service) scoreAmount(txAmount *decimal.Decimal, receiptAmount *decimal.Decimal) float64 {
	if txAmount == nil || receiptAmount == nil {
		return 0
	}

	txDec := *txAmount
	receiptDec := *receiptAmount

	if txDec.Equal(receiptDec) {
		return 1.0
	}

	if txDec.LessThan(receiptDec) {
		return 0
	}

	maxDiff := receiptDec.Mul(decimal.NewFromFloat(0.2))
	diff := txDec.Sub(receiptDec).Abs()

	if diff.GreaterThan(maxDiff) {
		return 0
	}

	ratio, _ := diff.Div(maxDiff).Float64()
	return 0.9 * (1.0 - ratio)
}


func (s *service) scoreDateMatch(txDate time.Time, receiptDate *time.Time) float64 {
	if receiptDate == nil || txDate.IsZero() {
		return 0.5 // neutral if no date
	}

	receiptTime := time.Date(receiptDate.Year(), receiptDate.Month(), receiptDate.Day(), 0, 0, 0, 0, time.UTC)

	d1 := time.Date(txDate.Year(), txDate.Month(), txDate.Day(), 0, 0, 0, 0, time.UTC)
	days := math.Abs(d1.Sub(receiptTime).Hours() / 24)

	if days >= 30 {
		return 0
	}

	return 1.0 - (days / 30.0)
}

func (s *service) scoreMerchant(txDesc *string, txMerchant *string, receiptMerchant *string) float64 {
	if receiptMerchant == nil || *receiptMerchant == "" {
		return 0.5 // neutral
	}

	receiptMerchantLower := strings.ToLower(*receiptMerchant)

	// check transaction merchant field first
	if txMerchant != nil && *txMerchant != "" {
		txMerchantLower := strings.ToLower(*txMerchant)
		if strings.Contains(txMerchantLower, receiptMerchantLower) ||
			strings.Contains(receiptMerchantLower, txMerchantLower) {
			return 0.9
		}
	}

	// fallback to description
	if txDesc != nil && *txDesc != "" {
		txDescLower := strings.ToLower(*txDesc)
		if strings.Contains(txDescLower, receiptMerchantLower) ||
			strings.Contains(receiptMerchantLower, txDescLower) {
			return 0.7
		}
	}

	return 0.3 // poor match
}

func (s *service) setLinkStatus(ctx context.Context, receiptID int64, status int16) error {
	// UpdateReceipt now takes individual parameters instead of a params struct
	_, err := s.queries.UpdateReceipt(ctx, sqlc.UpdateReceiptParams{
		Engine:         nil,
		ParseStatus:    nil,
		LinkStatus:     &status,
		MatchIds:       nil,
		Merchant:       nil,
		PurchaseDate:   nil,
		TotalAmount:    nil,
		Currency:       nil,
		TaxAmount:      nil,
		RawPayload:     nil,
		CanonicalData:  nil,
		ImageUrl:       nil,
		ImageSha256:    nil,
		Lat:            nil,
		Lon:            nil,
		LocationSource: nil,
		LocationLabel:  nil,
		ID:             receiptID,
	})
	return err
}

func (s *service) updateReceiptWithMatch(ctx context.Context, receiptID, transactionID int64, linkStatus int16, matchIDs []int64) error {
	// UpdateReceipt now takes individual parameters instead of a params struct
	_, err := s.queries.UpdateReceipt(ctx, sqlc.UpdateReceiptParams{
		Engine:         nil,
		ParseStatus:    nil,
		LinkStatus:     &linkStatus,
		MatchIds:       matchIDs,
		Merchant:       nil,
		PurchaseDate:   nil,
		TotalAmount:    nil,
		Currency:       nil,
		TaxAmount:      nil,
		RawPayload:     nil,
		CanonicalData:  nil,
		ImageUrl:       nil,
		ImageSha256:    nil,
		Lat:            nil,
		Lon:            nil,
		LocationSource: nil,
		LocationLabel:  nil,
		ID:             receiptID,
	})
	if err != nil {
		return err
	}

	// LinkTransactionToReceipt now takes individual parameters
	return s.queries.LinkTransactionToReceipt(ctx, sqlc.LinkTransactionToReceiptParams{
		ReceiptID:     receiptID,
		TransactionID: transactionID,
	})
}
*/
