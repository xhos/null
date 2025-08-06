package linking

import (
	sqlc "ariand/internal/db/sqlc"
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/money"
)

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
	transaction *sqlc.FindCandidateTransactionsRow
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

func (s *service) findCandidateTransactions(ctx context.Context, receipt sqlc.Receipt) ([]sqlc.FindCandidateTransactionsRow, error) {
	if receipt.TotalAmount == nil {
		return nil, nil
	}

	// search within 30 days and ±20% amount
	now := time.Now()
	startDate := now.AddDate(0, 0, -30)
	endDate := now.AddDate(0, 0, 1)
	targetDate := now

	if receipt.PurchaseDate != nil {
		targetDate = time.Date(int(receipt.PurchaseDate.Year), time.Month(receipt.PurchaseDate.Month), int(receipt.PurchaseDate.Day), 0, 0, 0, 0, time.UTC)
		startDate = targetDate.AddDate(0, 0, -30)
		endDate = targetDate.AddDate(0, 0, 30)
	}

	receiptAmount := s.moneyToDecimal(receipt.TotalAmount)
	minAmount := receiptAmount.Mul(decimal.NewFromFloat(0.8))
	maxAmount := receiptAmount.Mul(decimal.NewFromFloat(1.2))

	return s.queries.FindCandidateTransactions(ctx, sqlc.FindCandidateTransactionsParams{
		MinAmount:    minAmount,
		MaxAmount:    maxAmount,
		StartDate:    startDate,
		EndDate:      endDate,
		TargetAmount: receiptAmount,
		TargetDate:   targetDate,
	})
}

func (s *service) scoreMatches(candidates []sqlc.FindCandidateTransactionsRow, receipt sqlc.Receipt) []transactionMatch {
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

func (s *service) calculateScore(tx sqlc.FindCandidateTransactionsRow, receipt sqlc.Receipt) float64 {
	amountScore := s.scoreAmount(tx.TxAmount, receipt.TotalAmount)
	dateScore := s.scoreDateMatch(tx.TxDate, receipt.PurchaseDate)
	merchantScore := s.scoreMerchant(tx.TxDesc, tx.Merchant, receipt.Merchant)

	return amountScore*0.45 + dateScore*0.35 + merchantScore*0.2
}

func (s *service) scoreAmount(txAmount *money.Money, receiptAmount *money.Money) float64 {
	if txAmount == nil || receiptAmount == nil {
		return 0
	}

	txDec := s.moneyToDecimal(txAmount)
	receiptDec := s.moneyToDecimal(receiptAmount)

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

func (s *service) moneyToDecimal(m *money.Money) decimal.Decimal {
	if m == nil {
		return decimal.Zero
	}
	return decimal.NewFromFloat(float64(m.Units) + float64(m.Nanos)/1e9)
}

func (s *service) scoreDateMatch(txDate time.Time, receiptDate *date.Date) float64 {
	if receiptDate == nil || txDate.IsZero() {
		return 0.5 // neutral if no date
	}

	receiptTime := time.Date(int(receiptDate.Year), time.Month(receiptDate.Month), int(receiptDate.Day), 0, 0, 0, 0, time.UTC)

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
	_, err := s.queries.UpdateReceipt(ctx, sqlc.UpdateReceiptParams{
		ID:         receiptID,
		LinkStatus: &status,
	})
	return err
}

func (s *service) updateReceiptWithMatch(ctx context.Context, receiptID, transactionID int64, linkStatus int16, matchIDs []int64) error {
	_, err := s.queries.UpdateReceipt(ctx, sqlc.UpdateReceiptParams{
		ID:         receiptID,
		LinkStatus: &linkStatus,
		MatchIds:   matchIDs,
	})
	if err != nil {
		return err
	}

	return s.queries.LinkTransactionToReceipt(ctx, sqlc.LinkTransactionToReceiptParams{
		TransactionID: transactionID,
		ReceiptID:     receiptID,
	})
}
