package service

import (
	sqlc "ariand/internal/db/sqlc"
	"ariand/internal/linking"
	"ariand/internal/receiptparser"
	"ariand/internal/storage"
	"bytes"
	"context"
	"database/sql"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ReceiptService interface {
	ListForUser(ctx context.Context, userID uuid.UUID) ([]sqlc.Receipt, error)
	GetForUser(ctx context.Context, params sqlc.GetReceiptForUserParams) (*sqlc.Receipt, error)
	Create(ctx context.Context, params sqlc.CreateReceiptParams) (*sqlc.Receipt, error)
	Update(ctx context.Context, params sqlc.UpdateReceiptParams) error
	DeleteForUser(ctx context.Context, params sqlc.DeleteReceiptForUserParams) error

	ListItemsForReceipt(ctx context.Context, receiptID int64) ([]sqlc.ReceiptItem, error)
	GetItem(ctx context.Context, id int64) (*sqlc.ReceiptItem, error)
	CreateItem(ctx context.Context, params sqlc.CreateReceiptItemParams) (*sqlc.ReceiptItem, error)
	UpdateItem(ctx context.Context, params sqlc.UpdateReceiptItemParams) (*sqlc.ReceiptItem, error)
	DeleteItem(ctx context.Context, id int64) error
	BulkCreateItems(ctx context.Context, items []sqlc.BulkCreateReceiptItemsParams) error
	DeleteItemsByReceipt(ctx context.Context, receiptID int64) error

	GetUnlinked(ctx context.Context, limit *int32) ([]sqlc.GetUnlinkedReceiptsRow, error)
	GetMatchCandidates(ctx context.Context) ([]sqlc.GetReceiptMatchCandidatesRow, error)

	UploadReceipt(ctx context.Context, userID uuid.UUID, imageData []byte, provider string) (*sqlc.Receipt, error)
	ParseReceipt(ctx context.Context, receiptID int64, provider string) (*sqlc.Receipt, error)
	ConfirmReceipt(ctx context.Context, receiptID int64) error
	SearchReceipts(ctx context.Context, userID uuid.UUID, query string, limit *int32) ([]sqlc.Receipt, error)
	GetReceiptsByTransaction(ctx context.Context, transactionID int64) ([]sqlc.Receipt, error)
}

type receiptSvc struct {
	queries *sqlc.Queries
	parser  receiptparser.Client
	storage storage.Storage
	linking linking.Service
	log     *log.Logger
}

func newReceiptSvc(queries *sqlc.Queries, parser receiptparser.Client, storage storage.Storage, lg *log.Logger) ReceiptService {
	return &receiptSvc{
		queries: queries,
		parser:  parser,
		storage: storage,
		linking: linking.NewService(queries),
		log:     lg,
	}
}

func (s *receiptSvc) ListForUser(ctx context.Context, userID uuid.UUID) ([]sqlc.Receipt, error) {
	receipts, err := s.queries.ListReceiptsForUser(ctx, userID)
	if err != nil {
		return nil, wrapErr("ReceiptService.ListForUser", err)
	}
	return receipts, nil
}

func (s *receiptSvc) GetForUser(ctx context.Context, params sqlc.GetReceiptForUserParams) (*sqlc.Receipt, error) {
	receipt, err := s.queries.GetReceiptForUser(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("ReceiptService.GetForUser", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("ReceiptService.GetForUser", err)
	}
	return &receipt, nil
}

func (s *receiptSvc) Create(ctx context.Context, params sqlc.CreateReceiptParams) (*sqlc.Receipt, error) {
	receipt, err := s.queries.CreateReceipt(ctx, params)
	if err != nil {
		return nil, wrapErr("ReceiptService.Create", err)
	}
	return &receipt, nil
}

func (s *receiptSvc) Update(ctx context.Context, params sqlc.UpdateReceiptParams) error {
	_, err := s.queries.UpdateReceipt(ctx, params)
	if err != nil {
		return wrapErr("ReceiptService.Update", err)
	}
	return nil
}

func (s *receiptSvc) DeleteForUser(ctx context.Context, params sqlc.DeleteReceiptForUserParams) error {
	_, err := s.queries.DeleteReceiptForUser(ctx, params)
	if err != nil {
		return wrapErr("ReceiptService.DeleteForUser", err)
	}
	return nil
}

func (s *receiptSvc) ListItemsForReceipt(ctx context.Context, receiptID int64) ([]sqlc.ReceiptItem, error) {
	items, err := s.queries.ListReceiptItemsForReceipt(ctx, receiptID)
	if err != nil {
		return nil, wrapErr("ReceiptService.ListItemsForReceipt", err)
	}
	return items, nil
}

func (s *receiptSvc) GetItem(ctx context.Context, id int64) (*sqlc.ReceiptItem, error) {
	item, err := s.queries.GetReceiptItem(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("ReceiptService.GetItem", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("ReceiptService.GetItem", err)
	}
	return &item, nil
}

func (s *receiptSvc) CreateItem(ctx context.Context, params sqlc.CreateReceiptItemParams) (*sqlc.ReceiptItem, error) {
	item, err := s.queries.CreateReceiptItem(ctx, params)
	if err != nil {
		return nil, wrapErr("ReceiptService.CreateItem", err)
	}
	return &item, nil
}

func (s *receiptSvc) UpdateItem(ctx context.Context, params sqlc.UpdateReceiptItemParams) (*sqlc.ReceiptItem, error) {
	item, err := s.queries.UpdateReceiptItem(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("ReceiptService.UpdateItem", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("ReceiptService.UpdateItem", err)
	}
	return &item, nil
}

func (s *receiptSvc) DeleteItem(ctx context.Context, id int64) error {
	_, err := s.queries.DeleteReceiptItem(ctx, id)
	if err != nil {
		return wrapErr("ReceiptService.DeleteItem", err)
	}
	return nil
}

func (s *receiptSvc) BulkCreateItems(ctx context.Context, items []sqlc.BulkCreateReceiptItemsParams) error {
	_, err := s.queries.BulkCreateReceiptItems(ctx, items)
	if err != nil {
		return wrapErr("ReceiptService.BulkCreateItems", err)
	}
	return nil
}

func (s *receiptSvc) DeleteItemsByReceipt(ctx context.Context, receiptID int64) error {
	_, err := s.queries.DeleteReceiptItemsByReceipt(ctx, receiptID)
	if err != nil {
		return wrapErr("ReceiptService.DeleteItemsByReceipt", err)
	}
	return nil
}

func (s *receiptSvc) GetUnlinked(ctx context.Context, limit *int32) ([]sqlc.GetUnlinkedReceiptsRow, error) {
	receipts, err := s.queries.GetUnlinkedReceipts(ctx, limit)
	if err != nil {
		return nil, wrapErr("ReceiptService.GetUnlinked", err)
	}
	return receipts, nil
}

func (s *receiptSvc) GetMatchCandidates(ctx context.Context) ([]sqlc.GetReceiptMatchCandidatesRow, error) {
	candidates, err := s.queries.GetReceiptMatchCandidates(ctx)
	if err != nil {
		return nil, wrapErr("ReceiptService.GetMatchCandidates", err)
	}
	return candidates, nil
}

func (s *receiptSvc) UploadReceipt(ctx context.Context, userID uuid.UUID, imageData []byte, provider string) (*sqlc.Receipt, error) {
	// store image first
	imageURL, imageHash, err := s.storage.Store(imageData, "receipt.jpg")
	if err != nil {
		return nil, wrapErr("ReceiptService.UploadReceipt", err)
	}

	// create receipt record with pending status
	parseStatus := int16(1) // pending
	linkStatus := int16(1)  // unlinked

	params := sqlc.CreateReceiptParams{
		Engine:      1,
		ParseStatus: &parseStatus,
		LinkStatus:  &linkStatus,
		ImageUrl:    &imageURL,
		ImageSha256: imageHash,
	}

	receipt, err := s.queries.CreateReceipt(ctx, params)
	if err != nil {
		// cleanup stored image on failure
		s.storage.Delete(imageURL)
		return nil, wrapErr("ReceiptService.UploadReceipt", err)
	}

	return &receipt, nil
}

func (s *receiptSvc) ParseReceipt(ctx context.Context, receiptID int64, provider string) (*sqlc.Receipt, error) {
	// get receipt to access stored image
	receipt, err := s.queries.GetReceipt(ctx, receiptID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("ReceiptService.ParseReceipt", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("ReceiptService.ParseReceipt", err)
	}

	if receipt.ImageUrl == nil {
		return nil, wrapErr("ReceiptService.ParseReceipt", errors.New("no image stored for receipt"))
	}

	// get image data from storage
	imageData, err := s.storage.Get(*receipt.ImageUrl)
	if err != nil {
		return nil, wrapErr("ReceiptService.ParseReceipt", err)
	}

	// parse with microservice
	parsedReceipt, err := s.parser.Parse(ctx,
		bytes.NewReader(imageData),
		"receipt.jpg",
		"image/jpeg",
		nil,
	)

	// update receipt with parse results
	parseStatus := int16(3) // failed
	updateParams := sqlc.UpdateReceiptParams{
		ID:          receiptID,
		ParseStatus: &parseStatus,
	}

	if err != nil {
		s.log.Warn("failed to parse receipt", "receiptID", receiptID, "error", err)
	} else {
		successStatus := int16(2) // success
		updateParams.ParseStatus = &successStatus

		if parsedReceipt.Merchant != nil && *parsedReceipt.Merchant != "" {
			updateParams.Merchant = parsedReceipt.Merchant
		}

		if parsedReceipt.TotalAmount != nil {
			totalAmountDecimal := decimal.NewFromFloat(float64(parsedReceipt.TotalAmount.Units) + float64(parsedReceipt.TotalAmount.Nanos)/1e9)
			updateParams.TotalAmount = &totalAmountDecimal
			updateParams.Currency = &parsedReceipt.TotalAmount.CurrencyCode
		}

		// create receipt items
		if len(parsedReceipt.Items) > 0 {
			// delete existing items first
			s.DeleteItemsByReceipt(ctx, receiptID)

			var itemParams []sqlc.BulkCreateReceiptItemsParams
			for i, item := range parsedReceipt.Items {
				qtyDecimal := decimal.NewFromFloat(item.Quantity)
				lineNo := int32(i + 1)

				itemParams = append(itemParams, sqlc.BulkCreateReceiptItemsParams{
					ReceiptID: receiptID,
					LineNo:    &lineNo,
					Name:      item.Name,
					Qty:       &qtyDecimal,
					UnitPrice: item.UnitPrice,
					LineTotal: item.LineTotal,
				})
			}

			if err := s.BulkCreateItems(ctx, itemParams); err != nil {
				s.log.Warn("failed to create receipt items", "receiptID", receiptID, "error", err)
			}
		}
	}

	if err := s.Update(ctx, updateParams); err != nil {
		return nil, wrapErr("ReceiptService.ParseReceipt", err)
	}

	// attempt to link to transaction if parsing succeeded
	if err == nil {
		if linkErr := s.linking.LinkReceiptToTransaction(ctx, receiptID); linkErr != nil {
			s.log.Warn("failed to link receipt to transaction", "receiptID", receiptID, "error", linkErr)
		}
	}

	// return updated receipt
	updatedReceipt, err := s.queries.GetReceipt(ctx, receiptID)
	if err != nil {
		return nil, wrapErr("ReceiptService.ParseReceipt", err)
	}

	return &updatedReceipt, nil
}

func (s *receiptSvc) SearchReceipts(ctx context.Context, userID uuid.UUID, query string, limit *int32) ([]sqlc.Receipt, error) {
	return nil, wrapErr("ReceiptService.SearchReceipts", ErrUnimplemented)
}

func (s *receiptSvc) ConfirmReceipt(ctx context.Context, receiptID int64) error {
	receipt, err := s.queries.GetReceipt(ctx, receiptID)
	if err != nil {
		return wrapErr("ReceiptService.ConfirmReceipt", err)
	}

	// delete stored image when receipt is confirmed
	if receipt.ImageUrl != nil {
		if err := s.storage.Delete(*receipt.ImageUrl); err != nil {
			s.log.Warn("failed to delete receipt image", "receiptID", receiptID, "url", *receipt.ImageUrl, "error", err)
		}

		// clear image fields
		updateParams := sqlc.UpdateReceiptParams{
			ID:          receiptID,
			ImageUrl:    new(string), // empty string
			ImageSha256: []byte{},
		}

		if err := s.Update(ctx, updateParams); err != nil {
			return wrapErr("ReceiptService.ConfirmReceipt", err)
		}
	}

	return nil
}

func (s *receiptSvc) GetReceiptsByTransaction(ctx context.Context, transactionID int64) ([]sqlc.Receipt, error) {
	return nil, wrapErr("ReceiptService.GetReceiptsByTransaction", ErrUnimplemented)
}
