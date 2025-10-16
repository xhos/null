package api

import (
	"ariand/internal/backup"
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"ariand/internal/service"
	"ariand/internal/types"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ==================== ERROR HANDLING ====================

// Error handling helper
func handleError(err error) error {
	if err == nil {
		return nil
	}

	if err == service.ErrNotFound {
		return status.Error(codes.NotFound, err.Error())
	}
	if err == service.ErrValidation {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if err == service.ErrUnimplemented {
		return status.Error(codes.Unimplemented, err.Error())
	}

	return status.Errorf(codes.Internal, "internal error: %v", err)
}

// Timestamp helpers - kept for backward compatibility where needed
func toProtoTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func fromProtoTimestamp(ts *timestamppb.Timestamp) time.Time {
	if ts == nil || !ts.IsValid() {
		return time.Time{}
	}
	return ts.AsTime()
}

// Date to timestamp conversion
func dateToProtoTimestamp(d time.Time) *timestamppb.Timestamp {
	// Convert date to beginning of day in UTC
	t := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	return timestamppb.New(t)
}

// Timestamp to date conversion
func timestampToDate(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// Convert *time.Time to *date.Date for protobuf
func timeToProtoDate(t *time.Time) *date.Date {
	if t == nil {
		return nil
	}
	return &date.Date{
		Year:  int32(t.Year()),
		Month: int32(t.Month()),
		Day:   int32(t.Day()),
	}
}

// ==================== MONEY CONVERSION HELPERS ====================

// Convert int64 to money.Money with currency (for dashboard aggregations)
func int64ToMoney(amount int64, currency string) *money.Money {
	if currency == "" {
		currency = "USD" // fallback default
	}

	return &money.Money{
		CurrencyCode: currency,
		Units:        amount,
		Nanos:        0,
	}
}

// Convert cents (int64) to money.Money with proper units and nanos
func centsToMoney(cents int64, currency string) *money.Money {
	if currency == "" {
		currency = "USD"
	}
	return &money.Money{
		CurrencyCode: currency,
		Units:        cents / 100,
		Nanos:        int32((cents % 100) * 10000000),
	}
}

// Convert float64 to money.Money with currency (for dashboard averages)
func float64ToMoney(amount float64, currency string) *money.Money {
	if currency == "" {
		currency = "USD" // fallback default
	}

	units := int64(amount)
	nanos := int32((amount - float64(units)) * 1e9)

	return &money.Money{
		CurrencyCode: currency,
		Units:        units,
		Nanos:        nanos,
	}
}

// Helper to get currency or default
// func getCurrencyOrDefault(currency *string) string {
// 	if currency != nil {
// 		return *currency
// 	}
// 	return "USD"
// }

// ==================== LEGACY HELPERS ====================

// UUID helpers
func parseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, status.Error(codes.InvalidArgument, "uuid cannot be empty")
	}

	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid uuid: %v", err)
	}

	return id, nil
}

// ==================== ACCOUNT PARAMETER BUILDERS ====================

// buildUpdateAccountParams creates sqlc params from proto request
func buildUpdateAccountParams(req *pb.UpdateAccountRequest) sqlc.UpdateAccountParams {
	params := sqlc.UpdateAccountParams{
		ID: req.GetId(),
	}

	if req.Name != nil {
		params.Name = req.Name
	}
	if req.Bank != nil {
		params.Bank = req.Bank
	}
	if req.AccountType != nil {
		accountType := int16(*req.AccountType)
		params.AccountType = &accountType
	}
	if req.Alias != nil {
		params.Alias = req.Alias
	}
	if req.AnchorDate != nil {
		t := req.AnchorDate.AsTime()
		params.AnchorDate = &t
	}
	if req.AnchorBalance != nil {
		balanceBytes, err := types.ToBytes(req.AnchorBalance)
		if err != nil {
			// This should be handled at the calling site, but for now we'll use zero bytes
			balanceBytes = []byte{}
		}
		params.AnchorBalance = balanceBytes
	}
	if req.MainCurrency != nil {
		params.MainCurrency = req.MainCurrency
	}
	if len(req.Colors) > 0 {
		params.Colors = req.Colors
	}

	return params
}

// ==================== ACCOUNT MAPPINGS ====================

func toProtoAccount(a *sqlc.Account) *pb.Account {
	if a == nil {
		return nil
	}

	return &pb.Account{
		Id:            a.ID,
		Name:          a.Name,
		Bank:          a.Bank,
		Type:          a.AccountType,
		Alias:         a.Alias,
		AnchorDate:    dateToProtoTimestamp(a.AnchorDate),
		AnchorBalance: types.Unwrap(a.AnchorBalance),
		Balance:       types.Unwrap(types.FromBytes(a.Balance)),
		MainCurrency:  a.MainCurrency,
		Colors:        a.Colors,
		CreatedAt:     toProtoTimestamp(&a.CreatedAt),
		UpdatedAt:     toProtoTimestamp(&a.UpdatedAt),
	}
}

func createAccountParamsFromProto(req *pb.CreateAccountRequest) (sqlc.CreateAccountParams, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return sqlc.CreateAccountParams{}, err
	}

	balanceBytes, err := types.ToBytes(req.GetAnchorBalance())
	if err != nil {
		return sqlc.CreateAccountParams{}, err
	}

	// Default to CAD if main_currency is empty
	mainCurrency := req.GetMainCurrency()
	if mainCurrency == "" {
		mainCurrency = "CAD"
	}

	// Default colors if not provided, validate if provided
	colors := req.GetColors()
	if len(colors) == 0 {
		colors = []string{"#1f2937", "#3b82f6", "#10b981"}
	} else if len(colors) != 3 {
		return sqlc.CreateAccountParams{}, fmt.Errorf("colors must be exactly 3 hex values, got %d", len(colors))
	}

	return sqlc.CreateAccountParams{
		OwnerID:       userID,
		Name:          req.GetName(),
		Bank:          req.GetBank(),
		AccountType:   int16(req.GetType()),
		Alias:         req.Alias,
		AnchorBalance: balanceBytes,
		MainCurrency:  mainCurrency,
		Colors:        colors,
	}, nil
}

// ==================== USER MAPPINGS ====================

func toProtoUser(u *sqlc.User) *pb.User {
	if u == nil {
		return nil
	}

	user := &pb.User{
		Id:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		CreatedAt:   toProtoTimestamp(&u.CreatedAt),
		UpdatedAt:   toProtoTimestamp(&u.UpdatedAt),
	}

	if u.DefaultAccountID != nil {
		user.DefaultAccountId = u.DefaultAccountID
	}

	return user
}

func createUserParamsFromProto(req *pb.CreateUserRequest) (sqlc.CreateUserParams, error) {
	userID, err := parseUUID(req.GetId())
	if err != nil {
		return sqlc.CreateUserParams{}, err
	}

	return sqlc.CreateUserParams{
		ID:          userID,
		Email:       req.GetEmail(),
		DisplayName: req.DisplayName,
	}, nil
}

func updateUserParamsFromProto(req *pb.UpdateUserRequest) (sqlc.UpdateUserParams, error) {
	userID, err := parseUUID(req.GetId())
	if err != nil {
		return sqlc.UpdateUserParams{}, err
	}

	return sqlc.UpdateUserParams{
		ID:               userID,
		Email:            req.Email,
		DisplayName:      req.DisplayName,
		DefaultAccountID: req.DefaultAccountId,
	}, nil
}

func setUserDefaultAccountParamsFromProto(req *pb.SetUserDefaultAccountRequest) (sqlc.SetUserDefaultAccountParams, error) {
	userID, err := parseUUID(req.GetId())
	if err != nil {
		return sqlc.SetUserDefaultAccountParams{}, err
	}

	return sqlc.SetUserDefaultAccountParams{
		ID:               userID,
		DefaultAccountID: req.GetDefaultAccountId(),
	}, nil
}

// ==================== CATEGORY MAPPINGS ====================

func toProtoCategory(c *sqlc.Category) *pb.Category {
	if c == nil {
		return nil
	}

	return &pb.Category{
		Id:    c.ID,
		Slug:  c.Slug,
		Color: c.Color,
	}
}

// ==================== TRANSACTION PARAMETER BUILDERS ====================

// buildListTransactionsParams creates sqlc params from proto request
func buildListTransactionsParams(userID uuid.UUID, req *pb.ListTransactionsRequest) sqlc.ListTransactionsParams {
	params := sqlc.ListTransactionsParams{
		UserID: userID,
		Limit:  req.Limit,
	}

	// handle cursor pagination
	if cursor := req.GetCursor(); cursor != nil {
		if cursor.Date != nil {
			cursorTime := fromProtoTimestamp(cursor.Date)
			params.CursorDate = &cursorTime
		}
		if cursor.Id != nil {
			params.CursorID = cursor.Id
		}
	}

	// handle filters
	if req.StartDate != nil {
		startTime := fromProtoTimestamp(req.StartDate)
		params.Start = &startTime
	}
	if req.EndDate != nil {
		endTime := fromProtoTimestamp(req.EndDate)
		params.End = &endTime
	}
	// Note: AmountMin/AmountMax removed - not used in queries
	if req.Direction != nil {
		direction := int16(*req.Direction)
		params.Direction = &direction
	}
	if len(req.AccountIds) > 0 {
		params.AccountIds = req.AccountIds
	}
	if len(req.Categories) > 0 {
		params.Categories = req.Categories
	}
	if req.MerchantQuery != nil {
		params.MerchantQ = req.MerchantQuery
	}
	if req.DescriptionQuery != nil {
		params.DescQ = req.DescriptionQuery
	}
	if req.Currency != nil {
		params.Currency = req.Currency
	}

	return params
}

// buildCreateTransactionParams creates sqlc params from proto request
func buildCreateTransactionParams(userID uuid.UUID, req *pb.CreateTransactionRequest) (sqlc.CreateTransactionParams, error) {
	txAmountBytes, err := types.ToBytes(req.TxAmount)
	if err != nil {
		return sqlc.CreateTransactionParams{}, err
	}

	categoryManuallySet := false
	merchantManuallySet := false

	params := sqlc.CreateTransactionParams{
		UserID:              userID,
		AccountID:           req.GetAccountId(),
		TxDate:              fromProtoTimestamp(req.TxDate),
		TxAmount:            txAmountBytes,
		TxDirection:         int16(req.Direction),
		TxDesc:              req.Description,
		Merchant:            req.Merchant,
		UserNotes:           req.UserNotes,
		CategoryManuallySet: &categoryManuallySet,
		MerchantManuallySet: &merchantManuallySet,
	}

	// if user provides category_id, mark it as manually set
	if req.CategoryId != nil {
		params.CategoryID = req.CategoryId
		manuallySet := true
		params.CategoryManuallySet = &manuallySet
	}

	// if user provides merchant, mark it as manually set
	if req.Merchant != nil {
		manuallySet := true
		params.MerchantManuallySet = &manuallySet
	}

	if req.ForeignAmount != nil {
		foreignAmountBytes, err := types.ToBytes(req.ForeignAmount)
		if err != nil {
			return sqlc.CreateTransactionParams{}, err
		}
		params.ForeignAmount = foreignAmountBytes
		if req.ExchangeRate != nil {
			params.ExchangeRate = req.ExchangeRate
		}
	}

	return params, nil
}

// buildUpdateTransactionParams creates sqlc params from proto request
func buildUpdateTransactionParams(userID uuid.UUID, req *pb.UpdateTransactionRequest) (sqlc.UpdateTransactionParams, error) {
	params := sqlc.UpdateTransactionParams{
		ID:     req.GetId(),
		UserID: userID,
	}

	// apply field mask updates
	if req.TxDate != nil {
		txTime := fromProtoTimestamp(req.TxDate)
		params.TxDate = &txTime
	}
	if req.TxAmount != nil {
		txAmountBytes, err := types.ToBytes(req.TxAmount)
		if err != nil {
			return sqlc.UpdateTransactionParams{}, err
		}
		params.TxAmount = txAmountBytes
	}
	if req.Direction != nil {
		direction := int16(*req.Direction)
		params.TxDirection = &direction
	}
	if req.Description != nil {
		params.TxDesc = req.Description
	}
	if req.Merchant != nil {
		params.Merchant = req.Merchant
		// if setting merchant, mark as manually set; if clearing, mark as not manually set
		if *req.Merchant != "" {
			manuallySet := true
			params.MerchantManuallySet = &manuallySet
		} else {
			manuallySet := false
			params.MerchantManuallySet = &manuallySet
		}
	}
	if req.UserNotes != nil {
		params.UserNotes = req.UserNotes
	}
	if req.CategoryId != nil {
		params.CategoryID = req.CategoryId
		// if setting category, mark as manually set; if clearing, mark as not manually set
		if *req.CategoryId > 0 {
			manuallySet := true
			params.CategoryManuallySet = &manuallySet
		} else {
			manuallySet := false
			params.CategoryManuallySet = &manuallySet
		}
	}
	if req.ForeignAmount != nil {
		foreignAmountBytes, err := types.ToBytes(req.ForeignAmount)
		if err != nil {
			return sqlc.UpdateTransactionParams{}, err
		}
		params.ForeignAmount = foreignAmountBytes
	}
	if req.ExchangeRate != nil {
		params.ExchangeRate = req.ExchangeRate
	}

	return params, nil
}

// buildNextCursor creates pagination cursor from last transaction
func buildNextCursor(transactions []sqlc.Transaction, limit *int32) *pb.Cursor {
	if len(transactions) == 0 || limit == nil || len(transactions) != int(*limit) {
		return nil
	}

	lastTx := transactions[len(transactions)-1]
	return &pb.Cursor{
		Date: toProtoTimestamp(&lastTx.TxDate),
		Id:   &lastTx.ID,
	}
}

// ==================== TRANSACTION MAPPINGS ====================

// transaction field extractor - extracts common transaction fields using reflection
type transactionFields struct {
	ID                  int64
	EmailID             *string
	AccountID           int64
	TxDate              time.Time
	TxAmount            *money.Money
	TxDirection         pb.TransactionDirection
	TxDesc              *string
	CategoryID          *int64
	CategoryManuallySet bool
	Merchant            *string
	MerchantManuallySet bool
	UserNotes           *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// extract fields from any transaction row type
func extractTransactionFields(row interface{}) *transactionFields {
	if row == nil {
		return nil
	}

	// all transaction row types have identical field names and types
	switch t := row.(type) {
	case *sqlc.Transaction:
		return &transactionFields{
			ID: t.ID, EmailID: t.EmailID, AccountID: t.AccountID, TxDate: t.TxDate,
			TxAmount: types.Unwrap(t.TxAmount), TxDirection: t.TxDirection, TxDesc: t.TxDesc,
			CategoryID: t.CategoryID, CategoryManuallySet: t.CategoryManuallySet,
			Merchant: t.Merchant, MerchantManuallySet: t.MerchantManuallySet,
			UserNotes: t.UserNotes, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	case *sqlc.FindCandidateTransactionsRow:
		return &transactionFields{
			ID: t.ID, EmailID: t.EmailID, AccountID: t.AccountID, TxDate: t.TxDate,
			TxAmount: types.Unwrap(t.TxAmount), TxDirection: t.TxDirection, TxDesc: t.TxDesc,
			CategoryID: t.CategoryID, CategoryManuallySet: t.CategoryManuallySet,
			Merchant: t.Merchant, MerchantManuallySet: t.MerchantManuallySet,
			UserNotes: t.UserNotes, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	default:
		return nil
	}
}

// unified transaction converter
func convertTransactionToProto(row interface{}) *pb.Transaction {
	fields := extractTransactionFields(row)
	if fields == nil {
		return nil
	}

	return &pb.Transaction{
		Id:                  fields.ID,
		TxDate:              toProtoTimestamp(&fields.TxDate),
		TxAmount:            fields.TxAmount,
		Direction:           fields.TxDirection,
		AccountId:           fields.AccountID,
		EmailId:             fields.EmailID,
		Description:         fields.TxDesc,
		CategoryId:          fields.CategoryID,
		CategoryManuallySet: fields.CategoryManuallySet,
		Merchant:            fields.Merchant,
		MerchantManuallySet: fields.MerchantManuallySet,
		UserNotes:           fields.UserNotes,
		CreatedAt:           toProtoTimestamp(&fields.CreatedAt),
		UpdatedAt:           toProtoTimestamp(&fields.UpdatedAt),
	}
}

// wrapper functions for type safety
func toProtoTransaction(t *sqlc.Transaction) *pb.Transaction {
	return convertTransactionToProto(t)
}

// ==================== RECEIPT PARAMETER BUILDERS ====================

// floatToMoney converts optional float64 to optional money.Money in CAD
func floatToMoney(f *float64) *money.Money {
	if f == nil {
		return nil
	}
	return &money.Money{
		CurrencyCode: "CAD",
		Units:        int64(*f),
		Nanos:        int32((*f - float64(int64(*f))) * 1e9),
	}
}

// buildReceiptItemParams creates sqlc params from proto receipt item request
func buildReceiptItemParams(req *pb.CreateReceiptItemRequest) (sqlc.CreateReceiptItemParams, error) {
	unitPriceBytes, err := types.ToBytes(floatToMoney(req.UnitPrice))
	if err != nil {
		return sqlc.CreateReceiptItemParams{}, err
	}

	lineTotalBytes, err := types.ToBytes(floatToMoney(req.LineTotal))
	if err != nil {
		return sqlc.CreateReceiptItemParams{}, err
	}

	var qty *int32
	if req.Qty != nil {
		qtyInt := int32(*req.Qty)
		qty = &qtyInt
	}

	return sqlc.CreateReceiptItemParams{
		ReceiptID: req.GetReceiptId(),
		Name:      req.GetName(),
		LineNo:    req.LineNo,
		Qty:       qty,
		UnitPrice: unitPriceBytes,
		LineTotal: lineTotalBytes,
		Sku:       req.Sku,
	}, nil
}

// buildUpdateReceiptItemParams creates sqlc params from proto update request
func buildUpdateReceiptItemParams(req *pb.UpdateReceiptItemRequest) (sqlc.UpdateReceiptItemParams, error) {
	unitPriceBytes, err := types.ToBytes(floatToMoney(req.UnitPrice))
	if err != nil {
		return sqlc.UpdateReceiptItemParams{}, err
	}

	lineTotalBytes, err := types.ToBytes(floatToMoney(req.LineTotal))
	if err != nil {
		return sqlc.UpdateReceiptItemParams{}, err
	}

	var qty *int32
	if req.Qty != nil {
		qtyInt := int32(*req.Qty)
		qty = &qtyInt
	}

	return sqlc.UpdateReceiptItemParams{
		ID:        req.GetId(),
		Name:      req.Name,
		LineNo:    req.LineNo,
		Qty:       qty,
		UnitPrice: unitPriceBytes,
		LineTotal: lineTotalBytes,
		Sku:       req.Sku,
	}, nil
}

// buildBulkCreateReceiptItemsParams converts proto items to sqlc params
func buildBulkCreateReceiptItemsParams(items []*pb.CreateReceiptItemRequest) []sqlc.BulkCreateReceiptItemsParams {
	params := make([]sqlc.BulkCreateReceiptItemsParams, len(items))
	for i, item := range items {
		var qty *int32
		if item.Qty != nil {
			qtyInt := int32(*item.Qty)
			qty = &qtyInt
		}

		params[i] = sqlc.BulkCreateReceiptItemsParams{
			ReceiptID: item.GetReceiptId(),
			Name:      item.GetName(),
			LineNo:    item.LineNo,
			Qty:       qty,
			UnitPrice: types.Wrap(floatToMoney(item.UnitPrice)),
			LineTotal: types.Wrap(floatToMoney(item.LineTotal)),
			Sku:       item.Sku,
		}
	}
	return params
}

// buildReceiptItemsResponse converts receipt items to proto with proper money handling
func buildReceiptItemsResponse(items []sqlc.ReceiptItem) []*pb.ReceiptItem {
	pbItems := make([]*pb.ReceiptItem, len(items))
	for i, item := range items {
		pbItems[i] = toProtoReceiptItem(&item)
	}
	return pbItems
}

// buildReceiptSummariesResponse converts unlinked receipts to summaries
func buildReceiptSummariesResponse(receipts []sqlc.Receipt) []*pb.ReceiptSummary {
	pbReceipts := make([]*pb.ReceiptSummary, len(receipts))
	for i, receipt := range receipts {
		pbReceipts[i] = &pb.ReceiptSummary{
			Id:          receipt.ID,
			Merchant:    receipt.Merchant,
			TotalAmount: types.Unwrap(receipt.TotalAmount),
			CreatedAt:   toProtoTimestamp(&receipt.CreatedAt),
		}
	}
	return pbReceipts
}

// buildMatchCandidatesResponse converts match candidates to proto
func buildMatchCandidatesResponse(candidates []sqlc.GetReceiptMatchCandidatesRow) []*pb.ReceiptMatchCandidate {
	pbCandidates := make([]*pb.ReceiptMatchCandidate, len(candidates))
	for i, candidate := range candidates {
		// convert the row to a receipt
		receipt := &sqlc.Receipt{
			ID:           candidate.ID,
			Merchant:     candidate.Merchant,
			PurchaseDate: candidate.PurchaseDate,
			TotalAmount:  candidate.TotalAmount,
		}

		pbCandidates[i] = &pb.ReceiptMatchCandidate{
			Receipt:          toProtoReceipt(receipt),
			PotentialMatches: candidate.PotentialMatches,
		}
	}
	return pbCandidates
}

// ==================== RECEIPT MAPPINGS ====================

func toProtoReceipt(r *sqlc.Receipt) *pb.Receipt {
	if r == nil {
		return nil
	}

	var rawPayload *string
	if len(r.RawPayload) > 0 {
		payload := string(r.RawPayload)
		rawPayload = &payload
	}

	var canonicalData *string
	if len(r.CanonicalData) > 0 {
		data := string(r.CanonicalData)
		canonicalData = &data
	}

	return &pb.Receipt{
		Id:             r.ID,
		Engine:         r.Engine,
		ParseStatus:    r.ParseStatus,
		LinkStatus:     r.LinkStatus,
		RawPayload:     rawPayload,
		CanonicalData:  canonicalData,
		Merchant:       r.Merchant,
		TotalAmount:    types.Unwrap(r.TotalAmount),
		TaxAmount:      types.Unwrap(r.TaxAmount),
		PurchaseDate:   timeToProtoDate(r.PurchaseDate),
		MatchIds:       r.MatchIds,
		ImageUrl:       r.ImageUrl,
		ImageSha256:    r.ImageSha256,
		Lat:            r.Lat,
		Lon:            r.Lon,
		LocationSource: r.LocationSource,
		LocationLabel:  r.LocationLabel,
		CreatedAt:      toProtoTimestamp(&r.CreatedAt),
		UpdatedAt:      toProtoTimestamp(&r.UpdatedAt),
	}
}

func toProtoReceiptItem(ri *sqlc.ReceiptItem) *pb.ReceiptItem {
	if ri == nil {
		return nil
	}

	var quantity float64
	if ri.Qty != nil {
		quantity = float64(*ri.Qty)
	}

	return &pb.ReceiptItem{
		Id:           ri.ID,
		ReceiptId:    ri.ReceiptID,
		Name:         ri.Name,
		LineNo:       ri.LineNo,
		Quantity:     quantity,
		UnitPrice:    types.Unwrap(ri.UnitPrice),
		LineTotal:    types.Unwrap(ri.LineTotal),
		Sku:          ri.Sku,
		CategoryHint: ri.CategoryHint,
		CreatedAt:    toProtoTimestamp(&ri.CreatedAt),
		UpdatedAt:    toProtoTimestamp(&ri.UpdatedAt),
	}
}

// ==================== DASHBOARD PARAMETER BUILDERS ====================

// buildDashboardSummaryParams creates sqlc params from proto request
func buildDashboardSummaryParams(userID uuid.UUID, req *pb.GetDashboardSummaryRequest) sqlc.GetDashboardSummaryParams {
	return sqlc.GetDashboardSummaryParams{
		UserID: userID,
		Start:  dateToTime(req.StartDate),
		End:    dateToTime(req.EndDate),
	}
}

// buildDashboardTrendsParams creates sqlc params from proto request
func buildDashboardTrendsParams(userID uuid.UUID, req *pb.GetTrendDataRequest) sqlc.GetDashboardTrendsParams {
	return sqlc.GetDashboardTrendsParams{
		UserID: userID,
		Start:  dateToTime(req.StartDate),
		End:    dateToTime(req.EndDate),
	}
}

// buildTopCategoriesParams creates sqlc params from proto request
func buildTopCategoriesParams(userID uuid.UUID, req *pb.GetTopCategoriesRequest) sqlc.GetTopCategoriesParams {
	return sqlc.GetTopCategoriesParams{
		UserID: userID,
		Start:  dateToTime(req.StartDate),
		End:    dateToTime(req.EndDate),
		Limit:  req.Limit,
	}
}

// buildTopMerchantsParams creates sqlc params from proto request
func buildTopMerchantsParams(userID uuid.UUID, req *pb.GetTopMerchantsRequest) sqlc.GetTopMerchantsParams {
	return sqlc.GetTopMerchantsParams{
		UserID: userID,
		Start:  dateToTime(req.StartDate),
		End:    dateToTime(req.EndDate),
		Limit:  req.Limit,
	}
}

// buildMonthlyComparisonParams creates sqlc params from proto request
func buildMonthlyComparisonParams(userID uuid.UUID, monthsBack int32) sqlc.GetMonthlyComparisonParams {
	end := time.Now()
	start := end.AddDate(0, -int(monthsBack), 0)

	return sqlc.GetMonthlyComparisonParams{
		UserID: userID,
		Start:  &start,
		End:    &end,
	}
}

// ==================== DASHBOARD MAPPINGS ====================

func toProtoTrendPoint(trend *sqlc.GetDashboardTrendsRow) *pb.TrendPoint {
	if trend == nil {
		return nil
	}

	// parse the date string to time.Time for conversion
	trendDate, _ := time.Parse("2006-01-02", trend.Date)
	return &pb.TrendPoint{
		Date:     timeToDate(trendDate),
		Income:   centsToMoney(trend.IncomeCents, "CAD"),
		Expenses: centsToMoney(trend.ExpenseCents, "CAD"),
	}
}

func toProtoTrendPointFromAccount(trend *sqlc.GetDashboardTrendsForAccountRow) *pb.TrendPoint {
	if trend == nil {
		return nil
	}

	// parse the date string to time.Time for conversion
	trendDate, _ := time.Parse("2006-01-02", trend.Date)
	return &pb.TrendPoint{
		Date:     timeToDate(trendDate),
		Income:   centsToMoney(trend.IncomeCents, "CAD"),
		Expenses: centsToMoney(trend.ExpenseCents, "CAD"),
	}
}

func toProtoMonthlyComparison(comp *sqlc.GetMonthlyComparisonRow) *pb.MonthlyComparison {
	if comp == nil {
		return nil
	}

	return &pb.MonthlyComparison{
		Month:    comp.Month,
		Income:   centsToMoney(comp.IncomeCents, "CAD"),
		Expenses: centsToMoney(comp.ExpenseCents, "CAD"),
		Net:      centsToMoney(comp.NetCents, "CAD"),
	}
}

func toProtoTopCategory(cat *sqlc.GetTopCategoriesRow) *pb.TopCategory {
	if cat == nil {
		return nil
	}

	return &pb.TopCategory{
		Slug:             cat.Slug,
		Color:            cat.Color,
		TransactionCount: cat.TransactionCount,
		TotalAmount:      centsToMoney(cat.TotalAmountCents, "CAD"),
	}
}

func toProtoTopMerchant(merchant *sqlc.GetTopMerchantsRow) *pb.TopMerchant {
	if merchant == nil {
		return nil
	}

	merchantName := ""
	if merchant.Merchant != nil {
		merchantName = *merchant.Merchant
	}

	return &pb.TopMerchant{
		Merchant:         merchantName,
		TransactionCount: merchant.TransactionCount,
		TotalAmount:      centsToMoney(merchant.TotalAmountCents, "CAD"),
		AvgAmount:        centsToMoney(merchant.AvgAmountCents, "CAD"),
	}
}

func toProtoAccountBalance(account *sqlc.Account) *pb.AccountBalance {
	if account == nil {
		return nil
	}

	// use anchor balance as placeholders for current balance
	currentBalance := types.Unwrap(account.AnchorBalance)
	currency := "CAD" // default currency
	if currentBalance != nil {
		currency = currentBalance.CurrencyCode
	}

	return &pb.AccountBalance{
		Id:             account.ID,
		Name:           account.Name,
		AccountType:    pb.AccountType(account.AccountType),
		CurrentBalance: currentBalance,
		Currency:       currency,
	}
}

func toProtoDashboardSummary(summary *sqlc.GetDashboardSummaryRow) *pb.DashboardSummary {
	if summary == nil {
		return nil
	}

	return &pb.DashboardSummary{
		TotalAccounts:             summary.TotalAccounts,
		TotalTransactions:         summary.TotalTransactions,
		TotalIncome:               centsToMoney(summary.TotalIncomeCents, "CAD"),
		TotalExpenses:             centsToMoney(summary.TotalExpenseCents, "CAD"),
		UncategorizedTransactions: summary.UncategorizedTransactions,
	}
}

func toProtoDashboardSummaryFromAccount(summary *sqlc.GetDashboardSummaryForAccountRow) *pb.DashboardSummary {
	if summary == nil {
		return nil
	}

	return &pb.DashboardSummary{
		TotalAccounts:             summary.TotalAccounts,
		TotalTransactions:         summary.TotalTransactions,
		TotalIncome:               centsToMoney(summary.TotalIncomeCents, "CAD"),
		TotalExpenses:             centsToMoney(summary.TotalExpenseCents, "CAD"),
		UncategorizedTransactions: summary.UncategorizedTransactions,
	}
}

// ==================== HELPER FUNCTIONS ====================
// helper function to convert google.type.Date to time.Time
func dateToTime(d *date.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := time.Date(int(d.Year), time.Month(d.Month), int(d.Day), 0, 0, 0, 0, time.UTC)
	return &t
}

// helper function to convert time.Time to google.type.Date
func timeToDate(t time.Time) *date.Date {
	if t.IsZero() {
		return nil
	}
	return &date.Date{
		Year:  int32(t.Year()),
		Month: int32(t.Month()),
		Day:   int32(t.Day()),
	}
}

// ==================== PORTABILITY MAPPINGS ====================

func backupToProto(b *backup.Backup) *pb.Backup {
	if b == nil {
		return nil
	}

	protoCategories := make([]*pb.CategoryData, len(b.Categories))
	for i, cat := range b.Categories {
		protoCategories[i] = &pb.CategoryData{
			Slug:  cat.Slug,
			Color: cat.Color,
		}
	}

	protoAccounts := make([]*pb.AccountData, len(b.Accounts))
	for i, acc := range b.Accounts {
		protoAccounts[i] = &pb.AccountData{
			Name:          acc.Name,
			Bank:          acc.Bank,
			AccountType:   acc.AccountType,
			Alias:         acc.Alias,
			AnchorDate:    toProtoTimestamp(acc.AnchorDate),
			AnchorBalance: acc.AnchorBalance,
			MainCurrency:  acc.MainCurrency,
			Colors:        acc.Colors,
		}
	}

	protoTransactions := make([]*pb.TransactionData, len(b.Transactions))
	for i, tx := range b.Transactions {
		protoTransactions[i] = &pb.TransactionData{
			AccountName:   tx.AccountName,
			TxDate:        timestamppb.New(tx.TxDate),
			TxAmount:      tx.TxAmount,
			TxDirection:   tx.TxDirection,
			TxDesc:        tx.TxDesc,
			BalanceAfter:  tx.BalanceAfter,
			Merchant:      tx.Merchant,
			CategorySlug:  tx.CategorySlug,
			UserNotes:     tx.UserNotes,
			ForeignAmount: tx.ForeignAmount,
			ExchangeRate:  tx.ExchangeRate,
		}
	}

	protoRules := make([]*pb.RuleData, len(b.Rules))
	for i, rule := range b.Rules {
		conditionsJSON, _ := json.Marshal(rule.Conditions)
		protoRules[i] = &pb.RuleData{
			RuleName:       rule.RuleName,
			CategorySlug:   rule.CategorySlug,
			Merchant:       rule.Merchant,
			ConditionsJson: string(conditionsJSON),
			IsActive:       rule.IsActive,
			PriorityOrder:  rule.PriorityOrder,
			RuleSource:     rule.RuleSource,
		}
	}

	return &pb.Backup{
		Version:      b.Version,
		ExportedAt:   timestamppb.New(b.ExportedAt),
		Categories:   protoCategories,
		Accounts:     protoAccounts,
		Transactions: protoTransactions,
		Rules:        protoRules,
	}
}

func backupFromProto(pb *pb.Backup) *backup.Backup {
	if pb == nil {
		return nil
	}

	categories := make([]backup.CategoryData, len(pb.Categories))
	for i, cat := range pb.Categories {
		categories[i] = backup.CategoryData{
			Slug:  cat.Slug,
			Color: cat.Color,
		}
	}

	accounts := make([]backup.AccountData, len(pb.Accounts))
	for i, acc := range pb.Accounts {
		var anchorDate *time.Time
		if acc.AnchorDate != nil {
			t := acc.AnchorDate.AsTime()
			anchorDate = &t
		}

		accounts[i] = backup.AccountData{
			Name:          acc.Name,
			Bank:          acc.Bank,
			AccountType:   acc.AccountType,
			Alias:         acc.Alias,
			AnchorDate:    anchorDate,
			AnchorBalance: acc.AnchorBalance,
			MainCurrency:  acc.MainCurrency,
			Colors:        acc.Colors,
		}
	}

	transactions := make([]backup.TransactionData, len(pb.Transactions))
	for i, tx := range pb.Transactions {
		transactions[i] = backup.TransactionData{
			AccountName:   tx.AccountName,
			TxDate:        tx.TxDate.AsTime(),
			TxAmount:      tx.TxAmount,
			TxDirection:   tx.TxDirection,
			TxDesc:        tx.TxDesc,
			BalanceAfter:  tx.BalanceAfter,
			Merchant:      tx.Merchant,
			CategorySlug:  tx.CategorySlug,
			UserNotes:     tx.UserNotes,
			ForeignAmount: tx.ForeignAmount,
			ExchangeRate:  tx.ExchangeRate,
		}
	}

	rules := make([]backup.RuleData, len(pb.Rules))
	for i, rule := range pb.Rules {
		var conditions map[string]interface{}
		json.Unmarshal([]byte(rule.ConditionsJson), &conditions)

		rules[i] = backup.RuleData{
			RuleName:      rule.RuleName,
			CategorySlug:  rule.CategorySlug,
			Merchant:      rule.Merchant,
			Conditions:    conditions,
			IsActive:      rule.IsActive,
			PriorityOrder: rule.PriorityOrder,
			RuleSource:    rule.RuleSource,
		}
	}

	return &backup.Backup{
		Version:      pb.Version,
		ExportedAt:   pb.ExportedAt.AsTime(),
		Categories:   categories,
		Accounts:     accounts,
		Transactions: transactions,
		Rules:        rules,
	}
}
