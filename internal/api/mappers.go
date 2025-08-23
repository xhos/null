package api

import (
	"ariand/internal/api/middleware"
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"ariand/internal/service"
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ==================== AUTHENTICATION HELPERS ====================

// getUserFromContext extracts authenticated user from context
func getUserFromContext(ctx context.Context) (uuid.UUID, error) {
	user, ok := ctx.Value(middleware.UserContextKey).(*middleware.User)
	if !ok {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	userID, err := parseUUID(user.ID)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user ID"))
	}

	return userID, nil
}

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
func dateToProtoTimestamp(d *date.Date) *timestamppb.Timestamp {
	if d == nil {
		return nil
	}
	t := time.Date(int(d.Year), time.Month(d.Month), int(d.Day), 0, 0, 0, 0, time.UTC)
	return timestamppb.New(t)
}

// Timestamp to date conversion
func timestampToDate(ts *timestamppb.Timestamp) *date.Date {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &date.Date{
		Year:  int32(t.Year()),
		Month: int32(t.Month()),
		Day:   int32(t.Day()),
	}
}

// Money helpers - kept for backward compatibility
func decimalToMoney(val decimal.Decimal, currency string) *money.Money {
	if currency == "" {
		currency = "CAD" // default currency
	}

	f, _ := val.Float64()
	units := int64(f)
	nanos := int32((f - float64(units)) * 1e9)

	return &money.Money{
		CurrencyCode: currency,
		Units:        units,
		Nanos:        nanos,
	}
}

func moneyToDecimal(m *money.Money) *decimal.Decimal {
	if m == nil {
		return nil
	}

	val := float64(m.Units) + float64(m.Nanos)/1e9
	result := decimal.NewFromFloat(val)
	return &result
}

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
		params.AnchorDate = &date.Date{
			Year:  int32(t.Year()),
			Month: int32(t.Month()),
			Day:   int32(t.Day()),
		}
	}
	if req.AnchorBalance != nil {
		balance := moneyToDecimal(req.AnchorBalance)
		currency := req.AnchorBalance.CurrencyCode
		params.AnchorBalance = balance
		params.AnchorCurrency = &currency
	}

	return params
}

// ==================== ACCOUNT MAPPINGS ====================

func toProtoAccount(a *sqlc.ListAccountsForUserRow) *pb.Account {
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
		AnchorBalance: a.AnchorBalance,
		CreatedAt:     toProtoTimestamp(&a.CreatedAt),
		UpdatedAt:     toProtoTimestamp(&a.UpdatedAt),
	}
}

func toProtoAccountFromGetRow(a *sqlc.GetAccountForUserRow) *pb.Account {
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
		AnchorBalance: a.AnchorBalance,
		CreatedAt:     toProtoTimestamp(&a.CreatedAt),
		UpdatedAt:     toProtoTimestamp(&a.UpdatedAt),
	}
}

func toProtoAccountFromModel(a *sqlc.Account) *pb.Account {
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
		AnchorBalance: a.AnchorBalance,
		CreatedAt:     toProtoTimestamp(&a.CreatedAt),
		UpdatedAt:     toProtoTimestamp(&a.UpdatedAt),
	}
}

func createAccountParamsFromProto(req *pb.CreateAccountRequest) (sqlc.CreateAccountParams, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return sqlc.CreateAccountParams{}, err
	}

	currency := ""
	var balance decimal.Decimal
	if req.GetAnchorBalance() != nil {
		currency = req.GetAnchorBalance().CurrencyCode
		if dec := moneyToDecimal(req.GetAnchorBalance()); dec != nil {
			balance = *dec
		}
	}

	return sqlc.CreateAccountParams{
		OwnerID:        userID,
		Name:           req.GetName(),
		Bank:           req.GetBank(),
		AccountType:    int16(req.GetType()),
		Alias:          req.Alias,
		AnchorBalance:  balance,
		AnchorCurrency: currency,
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
		Id:        c.ID,
		Slug:      c.Slug,
		Label:     c.Label,
		Color:     c.Color,
		CreatedAt: toProtoTimestamp(&c.CreatedAt),
		UpdatedAt: toProtoTimestamp(&c.UpdatedAt),
	}
}

// ==================== TRANSACTION PARAMETER BUILDERS ====================

// buildListTransactionsParams creates sqlc params from proto request
func buildListTransactionsParams(userID uuid.UUID, req *pb.ListTransactionsRequest) sqlc.ListTransactionsForUserParams {
	params := sqlc.ListTransactionsForUserParams{
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
	if req.AmountMin != nil {
		params.AmountMin = moneyToDecimal(req.AmountMin)
	}
	if req.AmountMax != nil {
		params.AmountMax = moneyToDecimal(req.AmountMax)
	}
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
func buildCreateTransactionParams(userID uuid.UUID, req *pb.CreateTransactionRequest) sqlc.CreateTransactionForUserParams {
	params := sqlc.CreateTransactionForUserParams{
		UserID:      userID,
		AccountID:   req.GetAccountId(),
		TxDate:      fromProtoTimestamp(req.TxDate),
		TxAmount:    *moneyToDecimal(req.TxAmount),
		TxDirection: int16(req.Direction),
		TxDesc:      req.Description,
		Merchant:    req.Merchant,
		UserNotes:   req.UserNotes,
	}

	if req.CategoryId != nil {
		params.CategoryID = req.CategoryId
	}

	if req.ForeignAmount != nil {
		params.ForeignAmount = moneyToDecimal(req.ForeignAmount)
		if req.ExchangeRate != nil {
			exchangeRate := decimal.NewFromFloat(*req.ExchangeRate)
			params.ExchangeRate = &exchangeRate
		}
	}

	return params
}

// buildUpdateTransactionParams creates sqlc params from proto request
func buildUpdateTransactionParams(userID uuid.UUID, req *pb.UpdateTransactionRequest) sqlc.UpdateTransactionParams {
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
		amount := moneyToDecimal(req.TxAmount)
		params.TxAmount = amount
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
	}
	if req.UserNotes != nil {
		params.UserNotes = req.UserNotes
	}
	if req.CategoryId != nil {
		params.CategoryID = req.CategoryId
	}
	if req.ForeignAmount != nil {
		params.ForeignAmount = moneyToDecimal(req.ForeignAmount)
	}
	if req.ExchangeRate != nil {
		exchangeRate := decimal.NewFromFloat(*req.ExchangeRate)
		params.ExchangeRate = &exchangeRate
	}

	return params
}

// buildNextCursor creates pagination cursor from last transaction
func buildNextCursor(transactions []sqlc.ListTransactionsForUserRow, limit *int32) *pb.Cursor {
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
	ID          int64
	EmailID     *string
	AccountID   int64
	TxDate      time.Time
	TxAmount    *money.Money
	TxDirection pb.TransactionDirection
	TxDesc      *string
	CategoryID  *int64
	CatStatus   pb.CategorizationStatus
	Merchant    *string
	UserNotes   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// extract fields from any transaction row type
func extractTransactionFields(row interface{}) *transactionFields {
	if row == nil {
		return nil
	}

	// all transaction row types have identical field names and types
	switch t := row.(type) {
	case *sqlc.GetTransactionForUserRow:
		return &transactionFields{
			ID: t.ID, EmailID: t.EmailID, AccountID: t.AccountID, TxDate: t.TxDate,
			TxAmount: t.TxAmount, TxDirection: t.TxDirection, TxDesc: t.TxDesc,
			CategoryID: t.CategoryID, CatStatus: t.CatStatus, Merchant: t.Merchant,
			UserNotes: t.UserNotes, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	case *sqlc.ListTransactionsForUserRow:
		return &transactionFields{
			ID: t.ID, EmailID: t.EmailID, AccountID: t.AccountID, TxDate: t.TxDate,
			TxAmount: t.TxAmount, TxDirection: t.TxDirection, TxDesc: t.TxDesc,
			CategoryID: t.CategoryID, CatStatus: t.CatStatus, Merchant: t.Merchant,
			UserNotes: t.UserNotes, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		}
	case *sqlc.FindCandidateTransactionsForUserRow:
		return &transactionFields{
			ID: t.ID, EmailID: t.EmailID, AccountID: t.AccountID, TxDate: t.TxDate,
			TxAmount: t.TxAmount, TxDirection: t.TxDirection, TxDesc: t.TxDesc,
			CategoryID: t.CategoryID, CatStatus: t.CatStatus, Merchant: t.Merchant,
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
		Id:                   fields.ID,
		TxDate:               toProtoTimestamp(&fields.TxDate),
		TxAmount:             fields.TxAmount,
		Direction:            fields.TxDirection,
		AccountId:            fields.AccountID,
		EmailId:              fields.EmailID,
		Description:          fields.TxDesc,
		CategoryId:           fields.CategoryID,
		CategorizationStatus: fields.CatStatus,
		Merchant:             fields.Merchant,
		UserNotes:            fields.UserNotes,
		CreatedAt:            toProtoTimestamp(&fields.CreatedAt),
		UpdatedAt:            toProtoTimestamp(&fields.UpdatedAt),
	}
}

// wrapper functions for type safety
func toProtoTransactionFromGetRow(t *sqlc.GetTransactionForUserRow) *pb.Transaction {
	return convertTransactionToProto(t)
}

func toProtoTransactionFromListRow(t *sqlc.ListTransactionsForUserRow) *pb.Transaction {
	return convertTransactionToProto(t)
}

func toProtoTransactionFromFindRow(t *sqlc.FindCandidateTransactionsForUserRow) *pb.Transaction {
	return convertTransactionToProto(t)
}

// ==================== RECEIPT PARAMETER BUILDERS ====================

// floatToDecimal converts optional float64 to optional decimal.Decimal
func floatToDecimal(f *float64) *decimal.Decimal {
	if f == nil {
		return nil
	}
	result := decimal.NewFromFloat(*f)
	return &result
}

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
func buildReceiptItemParams(req *pb.CreateReceiptItemRequest) sqlc.CreateReceiptItemParams {
	return sqlc.CreateReceiptItemParams{
		ReceiptID: req.GetReceiptId(),
		Name:      req.GetName(),
		LineNo:    req.LineNo,
		Qty:       floatToDecimal(req.Qty),
		UnitPrice: floatToDecimal(req.UnitPrice), // convert float64 to decimal
		LineTotal: floatToDecimal(req.LineTotal), // convert float64 to decimal
		Sku:       req.Sku,
	}
}

// buildUpdateReceiptItemParams creates sqlc params from proto update request
func buildUpdateReceiptItemParams(req *pb.UpdateReceiptItemRequest) sqlc.UpdateReceiptItemParams {
	return sqlc.UpdateReceiptItemParams{
		ID:        req.GetId(),
		Name:      req.Name,
		LineNo:    req.LineNo,
		Qty:       floatToDecimal(req.Qty),
		UnitPrice: floatToDecimal(req.UnitPrice), // convert float64 to decimal
		LineTotal: floatToDecimal(req.LineTotal), // convert float64 to decimal
		Sku:       req.Sku,
	}
}

// buildBulkCreateReceiptItemsParams converts proto items to sqlc params
func buildBulkCreateReceiptItemsParams(items []*pb.CreateReceiptItemRequest) []sqlc.BulkCreateReceiptItemsParams {
	params := make([]sqlc.BulkCreateReceiptItemsParams, len(items))
	for i, item := range items {
		params[i] = sqlc.BulkCreateReceiptItemsParams{
			ReceiptID: item.GetReceiptId(),
			Name:      item.GetName(),
			LineNo:    item.LineNo,
			Qty:       floatToDecimal(item.Qty),
			UnitPrice: floatToMoney(item.UnitPrice), // convert float64 to money
			LineTotal: floatToMoney(item.LineTotal), // convert float64 to money
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
func buildReceiptSummariesResponse(receipts []sqlc.GetUnlinkedReceiptsRow) []*pb.ReceiptSummary {
	pbReceipts := make([]*pb.ReceiptSummary, len(receipts))
	for i, receipt := range receipts {
		pbReceipts[i] = &pb.ReceiptSummary{
			Id:          receipt.ID,
			Merchant:    receipt.Merchant,
			TotalAmount: receipt.TotalAmount,
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
			Currency:     candidate.Currency,
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
		TotalAmount:    r.TotalAmount,
		TaxAmount:      r.TaxAmount,
		PurchaseDate:   r.PurchaseDate,
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
		qtyFloat, _ := ri.Qty.Float64()
		quantity = qtyFloat
	}

	return &pb.ReceiptItem{
		Id:           ri.ID,
		ReceiptId:    ri.ReceiptID,
		Name:         ri.Name,
		LineNo:       ri.LineNo,
		Quantity:     quantity,
		UnitPrice:    ri.UnitPrice,
		LineTotal:    ri.LineTotal,
		Sku:          ri.Sku,
		CategoryHint: ri.CategoryHint,
		CreatedAt:    toProtoTimestamp(&ri.CreatedAt),
		UpdatedAt:    toProtoTimestamp(&ri.UpdatedAt),
	}
}

// ==================== DASHBOARD PARAMETER BUILDERS ====================

// buildDashboardSummaryParams creates sqlc params from proto request
func buildDashboardSummaryParams(userID uuid.UUID, req *pb.GetDashboardSummaryRequest) sqlc.GetDashboardSummaryForUserParams {
	return sqlc.GetDashboardSummaryForUserParams{
		UserID: userID,
		Start:  dateToTime(req.StartDate),
		End:    dateToTime(req.EndDate),
	}
}

// buildDashboardTrendsParams creates sqlc params from proto request
func buildDashboardTrendsParams(userID uuid.UUID, req *pb.GetTrendDataRequest) sqlc.GetDashboardTrendsForUserParams {
	return sqlc.GetDashboardTrendsForUserParams{
		UserID: userID,
		Start:  dateToTime(req.StartDate),
		End:    dateToTime(req.EndDate),
	}
}

// buildTopCategoriesParams creates sqlc params from proto request
func buildTopCategoriesParams(userID uuid.UUID, req *pb.GetTopCategoriesRequest) sqlc.GetTopCategoriesForUserParams {
	return sqlc.GetTopCategoriesForUserParams{
		UserID: userID,
		Start:  dateToTime(req.StartDate),
		End:    dateToTime(req.EndDate),
		Limit:  req.Limit,
	}
}

// buildTopMerchantsParams creates sqlc params from proto request
func buildTopMerchantsParams(userID uuid.UUID, req *pb.GetTopMerchantsRequest) sqlc.GetTopMerchantsForUserParams {
	return sqlc.GetTopMerchantsForUserParams{
		UserID: userID,
		Start:  dateToTime(req.StartDate),
		End:    dateToTime(req.EndDate),
		Limit:  req.Limit,
	}
}

// buildMonthlyComparisonParams creates sqlc params from proto request
func buildMonthlyComparisonParams(userID uuid.UUID, monthsBack int32) sqlc.GetMonthlyComparisonForUserParams {
	end := time.Now()
	start := end.AddDate(0, -int(monthsBack), 0)

	return sqlc.GetMonthlyComparisonForUserParams{
		UserID: userID,
		Start:  &start,
		End:    &end,
	}
}

// ==================== DASHBOARD MAPPINGS ====================

func toProtoTrendPoint(trend *sqlc.GetDashboardTrendsForUserRow) *pb.TrendPoint {
	if trend == nil {
		return nil
	}

	// parse the date string to time.Time for conversion
	trendDate, _ := time.Parse("2006-01-02", trend.Date)
	return &pb.TrendPoint{
		Date:     timeToDate(trendDate),
		Income:   decimalToMoney(decimal.NewFromInt(trend.Income), "CAD"),
		Expenses: decimalToMoney(decimal.NewFromInt(trend.Expenses), "CAD"),
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
		Income:   decimalToMoney(decimal.NewFromInt(trend.Income), "CAD"),
		Expenses: decimalToMoney(decimal.NewFromInt(trend.Expenses), "CAD"),
	}
}

func toProtoMonthlyComparison(comp *sqlc.GetMonthlyComparisonForUserRow) *pb.MonthlyComparison {
	if comp == nil {
		return nil
	}

	return &pb.MonthlyComparison{
		Month:    comp.Month,
		Income:   decimalToMoney(decimal.NewFromInt(comp.Income), "CAD"),
		Expenses: decimalToMoney(decimal.NewFromInt(comp.Expenses), "CAD"),
		Net:      decimalToMoney(decimal.NewFromInt(comp.Net), "CAD"),
	}
}

func toProtoTopCategory(cat *sqlc.GetTopCategoriesForUserRow) *pb.TopCategory {
	if cat == nil {
		return nil
	}

	return &pb.TopCategory{
		Slug:             cat.Slug,
		Label:            cat.Label,
		Color:            cat.Color,
		TransactionCount: cat.TransactionCount,
		TotalAmount:      decimalToMoney(decimal.NewFromInt(cat.TotalAmount), "CAD"),
	}
}

func toProtoTopMerchant(merchant *sqlc.GetTopMerchantsForUserRow) *pb.TopMerchant {
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
		TotalAmount:      decimalToMoney(decimal.NewFromInt(merchant.TotalAmount), "CAD"),
		AvgAmount:        decimalToMoney(decimal.NewFromFloat(merchant.AvgAmount), "CAD"),
	}
}

func toProtoAccountBalance(account *sqlc.ListAccountsForUserRow) *pb.AccountBalance {
	if account == nil {
		return nil
	}

	// use anchor balance and currency as placeholders for current balance
	var currentBalance *money.Money
	if account.AnchorBalance != nil {
		currentBalance = account.AnchorBalance
	} else {
		currentBalance = &money.Money{CurrencyCode: account.AnchorCurrency, Units: 0, Nanos: 0}
	}

	return &pb.AccountBalance{
		Id:             account.ID,
		Name:           account.Name,
		AccountType:    pb.AccountType(account.AccountType),
		CurrentBalance: currentBalance,
		Currency:       account.AnchorCurrency,
	}
}

func toProtoDashboardSummary(summary *sqlc.GetDashboardSummaryForUserRow) *pb.DashboardSummary {
	if summary == nil {
		return nil
	}

	return &pb.DashboardSummary{
		TotalAccounts:             summary.TotalAccounts,
		TotalTransactions:         summary.TotalTransactions,
		TotalIncome:               decimalToMoney(interfaceToDecimal(summary.TotalIncome), "CAD"),
		TotalExpenses:             decimalToMoney(interfaceToDecimal(summary.TotalExpenses), "CAD"),
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
		TotalIncome:               decimalToMoney(interfaceToDecimal(summary.TotalIncome), "CAD"),
		TotalExpenses:             decimalToMoney(interfaceToDecimal(summary.TotalExpenses), "CAD"),
		UncategorizedTransactions: summary.UncategorizedTransactions,
	}
}

// helper to safely convert interface{} to decimal.Decimal
func interfaceToDecimal(v interface{}) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	switch val := v.(type) {
	case decimal.Decimal:
		return val
	case float64:
		return decimal.NewFromFloat(val)
	case int64:
		return decimal.NewFromInt(val)
	default:
		return decimal.Zero
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
