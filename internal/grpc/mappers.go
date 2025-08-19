package grpc

import (
	sqlc "ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"ariand/internal/service"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

// ==================== ACCOUNT COLLABORATION MAPPINGS ====================

func toProtoAccountCollaborator(c *sqlc.ListAccountCollaboratorsRow) *pb.AccountCollaborator {
	if c == nil {
		return nil
	}

	user := &pb.User{
		Id:          c.ID.String(),
		Email:       c.Email,
		DisplayName: c.DisplayName,
	}

	return &pb.AccountCollaborator{
		User:    user,
		AddedAt: toProtoTimestamp(&c.AddedAt),
	}
}

func toProtoAccountCollaboration(c *sqlc.ListUserCollaborationsRow) *pb.AccountCollaboration {
	if c == nil {
		return nil
	}

	owner := &pb.User{
		Email:       c.OwnerEmail,
		DisplayName: c.OwnerName,
	}

	return &pb.AccountCollaboration{
		AccountId:   c.AccountID,
		AccountName: c.AccountName,
		Bank:        c.Bank,
		AddedAt:     toProtoTimestamp(&c.AddedAt),
		Owner:       owner,
	}
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

func createUserParamsFromProto(req *pb.CreateUserRequest) sqlc.CreateUserParams {
	return sqlc.CreateUserParams{
		Email:       req.GetEmail(),
		DisplayName: req.DisplayName,
	}
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

func toProtoCategoryFromUserRow(c *sqlc.ListCategoriesForUserRow) *pb.Category {
	if c == nil {
		return nil
	}

	category := &pb.Category{
		Id:        c.ID,
		Slug:      c.Slug,
		Label:     c.Label,
		Color:     c.Color,
		CreatedAt: toProtoTimestamp(&c.CreatedAt),
		UpdatedAt: toProtoTimestamp(&c.UpdatedAt),
	}

	category.UsageCount = &c.UserUsageCount

	return category
}

// ==================== TRANSACTION MAPPINGS ====================

func toProtoTransaction(t *sqlc.Transaction) *pb.Transaction {
	if t == nil {
		return nil
	}

	return &pb.Transaction{
		Id:                   t.ID,
		TxDate:               toProtoTimestamp(&t.TxDate),
		TxAmount:             t.TxAmount,
		Direction:            t.TxDirection,
		AccountId:            t.AccountID,
		EmailId:              t.EmailID,
		Description:          t.TxDesc,
		CategoryId:           t.CategoryID,
		CategorizationStatus: t.CatStatus,
		Merchant:             t.Merchant,
		UserNotes:            t.UserNotes,
		CreatedAt:            toProtoTimestamp(&t.CreatedAt),
		UpdatedAt:            toProtoTimestamp(&t.UpdatedAt),
	}
}

func toProtoTransactionFromGetRow(t *sqlc.GetTransactionForUserRow) *pb.Transaction {
	if t == nil {
		return nil
	}

	return &pb.Transaction{
		Id:                   t.ID,
		TxDate:               toProtoTimestamp(&t.TxDate),
		TxAmount:             t.TxAmount,
		Direction:            t.TxDirection,
		AccountId:            t.AccountID,
		EmailId:              t.EmailID,
		Description:          t.TxDesc,
		CategoryId:           t.CategoryID,
		CategorizationStatus: t.CatStatus,
		Merchant:             t.Merchant,
		UserNotes:            t.UserNotes,
		CreatedAt:            toProtoTimestamp(&t.CreatedAt),
		UpdatedAt:            toProtoTimestamp(&t.UpdatedAt),
	}
}

func toProtoTransactionFromListRow(t *sqlc.ListTransactionsForUserRow) *pb.Transaction {
	if t == nil {
		return nil
	}

	return &pb.Transaction{
		Id:                   t.ID,
		TxDate:               toProtoTimestamp(&t.TxDate),
		TxAmount:             t.TxAmount,
		Direction:            t.TxDirection,
		AccountId:            t.AccountID,
		EmailId:              t.EmailID,
		Description:          t.TxDesc,
		CategoryId:           t.CategoryID,
		CategorizationStatus: t.CatStatus,
		Merchant:             t.Merchant,
		UserNotes:            t.UserNotes,
		CreatedAt:            toProtoTimestamp(&t.CreatedAt),
		UpdatedAt:            toProtoTimestamp(&t.UpdatedAt),
	}
}

func toProtoTransactionFromFindRow(t *sqlc.FindCandidateTransactionsForUserRow) *pb.Transaction {
	if t == nil {
		return nil
	}

	return &pb.Transaction{
		Id:                   t.ID,
		TxDate:               toProtoTimestamp(&t.TxDate),
		TxAmount:             t.TxAmount,
		Direction:            t.TxDirection,
		AccountId:            t.AccountID,
		EmailId:              t.EmailID,
		Description:          t.TxDesc,
		CategoryId:           t.CategoryID,
		CategorizationStatus: t.CatStatus,
		Merchant:             t.Merchant,
		UserNotes:            t.UserNotes,
		CreatedAt:            toProtoTimestamp(&t.CreatedAt),
		UpdatedAt:            toProtoTimestamp(&t.UpdatedAt),
	}
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

// ==================== HELPER FUNCTIONS ====================
