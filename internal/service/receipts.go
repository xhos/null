package service

import (
	"ariand/internal/db/sqlc"
	"ariand/internal/receiptparser"
	"ariand/internal/storage"
	"ariand/internal/types"
	"bytes"
	"context"
	"database/sql"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/genproto/googleapis/type/money"
)

type ReceiptService interface {
	List(ctx context.Context, userID uuid.UUID) ([]sqlc.Receipt, error)
	Get(ctx context.Context, params sqlc.GetReceiptParams) (*sqlc.Receipt, error)
	Create(ctx context.Context, params sqlc.CreateReceiptParams) (*sqlc.Receipt, error)
	Update(ctx context.Context, params sqlc.UpdateReceiptParams) error
	Delete(ctx context.Context, params sqlc.DeleteReceiptParams) error

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
	ParseReceipt(ctx context.Context, userID uuid.UUID, receiptID int64, provider string) (*sqlc.Receipt, error)
	ConfirmReceipt(ctx context.Context, userID uuid.UUID, receiptID int64) error
	SearchReceipts(ctx context.Context, userID uuid.UUID, query string, limit *int32) ([]sqlc.Receipt, error)
	GetReceiptsByTransaction(ctx context.Context, transactionID int64) ([]sqlc.Receipt, error)
}

type receiptSvc struct {
	queries *sqlc.Queries
	parser  receiptparser.Client
	storage storage.Storage
	// linking linking.Service // temporarily disabled
	log *log.Logger
}

func newReceiptSvc(queries *sqlc.Queries, parser receiptparser.Client, storage storage.Storage, lg *log.Logger) ReceiptService {
	return &receiptSvc{
		queries: queries,
		parser:  parser,
		storage: storage,
		// linking: linking.NewService(queries), // temporarily disabled
		log: lg,
	}
}

func (s *receiptSvc) List(ctx context.Context, userID uuid.UUID) ([]sqlc.Receipt, error) {
	receipts, err := s.queries.ListReceipts(ctx, userID)
	if err != nil {
		return nil, wrapErr("ReceiptService.List", err)
	}
	return receipts, nil
}

func (s *receiptSvc) Get(ctx context.Context, params sqlc.GetReceiptParams) (*sqlc.Receipt, error) {
	receipt, err := s.queries.GetReceipt(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("ReceiptService.Get", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("ReceiptService.Get", err)
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

func (s *receiptSvc) Delete(ctx context.Context, params sqlc.DeleteReceiptParams) error {
	_, err := s.queries.DeleteReceipt(ctx, params)
	if err != nil {
		return wrapErr("ReceiptService.Delete", err)
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
	imageURL, imageHash, err := s.storage.Store(imageData, "receipt.jpg")
	if err != nil {
		return nil, wrapErr("ReceiptService.UploadReceipt", err)
	}

	pendingStatus := int16(1)
	unlinkedStatus := int16(1)

	params := sqlc.CreateReceiptParams{
		Engine:      1,
		ParseStatus: &pendingStatus,
		LinkStatus:  &unlinkedStatus,
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

func (s *receiptSvc) ParseReceipt(ctx context.Context, userID uuid.UUID, receiptID int64, provider string) (*sqlc.Receipt, error) {
	// get receipt to access stored image
	receipt, err := s.queries.GetReceipt(ctx, sqlc.GetReceiptParams{
		UserID: userID,
		ID:     receiptID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("ReceiptService.ParseReceipt", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("ReceiptService.ParseReceipt", err)
	}

	hasNoImage := receipt.ImageUrl == nil
	if hasNoImage {
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

	failedStatus := int16(3)
	updateParams := sqlc.UpdateReceiptParams{
		ID:          receiptID,
		ParseStatus: &failedStatus,
	}

	if err != nil {
		s.log.Warn("failed to parse receipt", "receiptID", receiptID, "error", err)
	} else {
		successStatus := int16(2)
		updateParams.ParseStatus = &successStatus

		if parsedReceipt.Merchant != nil && *parsedReceipt.Merchant != "" {
			updateParams.Merchant = parsedReceipt.Merchant
		}

		if parsedReceipt.TotalAmount != nil {
			totalAmount := &types.Money{
				Money: money.Money{
					CurrencyCode: parsedReceipt.TotalAmount.CurrencyCode,
					Units:        parsedReceipt.TotalAmount.Units,
					Nanos:        parsedReceipt.TotalAmount.Nanos,
				},
			}
			jsonBytes, err := totalAmount.Value()
			if err == nil {
				updateParams.TotalAmount = jsonBytes.([]byte)
			}
		}

		// create receipt items
		if len(parsedReceipt.Items) > 0 {
			if err := s.DeleteItemsByReceipt(ctx, receiptID); err != nil {
				s.log.Warn("failed to delete existing receipt items", "receiptID", receiptID, "error", err)
			}

			var itemParams []sqlc.BulkCreateReceiptItemsParams
			for i, item := range parsedReceipt.Items {
				lineNo := int32(i + 1)

				var unitPrice, lineTotal *types.Money
				if item.UnitPrice != nil {
					unitPrice = &types.Money{
						Money: money.Money{
							CurrencyCode: item.UnitPrice.CurrencyCode,
							Units:        item.UnitPrice.Units,
							Nanos:        item.UnitPrice.Nanos,
						},
					}
				}
				if item.LineTotal != nil {
					lineTotal = &types.Money{
						Money: money.Money{
							CurrencyCode: item.LineTotal.CurrencyCode,
							Units:        item.LineTotal.Units,
							Nanos:        item.LineTotal.Nanos,
						},
					}
				}

				qty := decimal.NewFromFloat(item.Quantity)

				itemParams = append(itemParams, sqlc.BulkCreateReceiptItemsParams{
					ReceiptID: receiptID,
					LineNo:    &lineNo,
					Name:      item.Name,
					Qty:       &qty,
					UnitPrice: unitPrice,
					LineTotal: lineTotal,
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
		// TODO: Re-enable linking when linking service is ready
		// if linkErr := s.linking.LinkReceiptToTransaction(ctx, receiptID); linkErr != nil {
		//	s.log.Warn("failed to link receipt to transaction", "receiptID", receiptID, "error", linkErr)
		// }
	}

	// return updated receipt
	updatedReceipt, err := s.queries.GetReceipt(ctx, sqlc.GetReceiptParams{
		UserID: userID,
		ID:     receiptID,
	})
	if err != nil {
		return nil, wrapErr("ReceiptService.ParseReceipt", err)
	}

	return &updatedReceipt, nil
}

func (s *receiptSvc) SearchReceipts(ctx context.Context, userID uuid.UUID, query string, limit *int32) ([]sqlc.Receipt, error) {
	return nil, wrapErr("ReceiptService.SearchReceipts", ErrUnimplemented)
}

func (s *receiptSvc) ConfirmReceipt(ctx context.Context, userID uuid.UUID, receiptID int64) error {
	receipt, err := s.queries.GetReceipt(ctx, sqlc.GetReceiptParams{
		UserID: userID,
		ID:     receiptID,
	})
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
