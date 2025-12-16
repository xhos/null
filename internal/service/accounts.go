package service

import (
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// centsToMoney converts cents to google.type.Money
func centsToMoney(cents int64, currency string) *money.Money {
	return &money.Money{
		CurrencyCode: currency,
		Units:        cents / 100,
		Nanos:        int32((cents % 100) * 10_000_000),
	}
}

// dateToProtoTimestamp converts time.Time (date only) to timestamppb.Timestamp
func dateToProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

// timeToProtoTimestamp converts a time pointer to timestamppb.Timestamp
func timeToProtoTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// rowToPbAccount converts sqlc row types to pb.Account
func rowToPbAccount(
	id int64,
	ownerID uuid.UUID,
	name, bank string,
	accountType pb.AccountType,
	alias *string,
	anchorDate time.Time,
	anchorBalanceCents int64,
	anchorCurrency, mainCurrency string,
	colors []string,
	createdAt, updatedAt time.Time,
	balanceCents int64,
	balanceCurrency string,
) *pb.Account {
	return &pb.Account{
		Id:            id,
		OwnerId:       ownerID.String(),
		Name:          name,
		Bank:          bank,
		Type:          accountType,
		Alias:         alias,
		AnchorDate:    dateToProtoTimestamp(anchorDate),
		AnchorBalance: centsToMoney(anchorBalanceCents, anchorCurrency),
		MainCurrency:  mainCurrency,
		Colors:        colors,
		CreatedAt:     timeToProtoTimestamp(&createdAt),
		UpdatedAt:     timeToProtoTimestamp(&updatedAt),
		Balance:       centsToMoney(balanceCents, balanceCurrency),
	}
}

type AccountService interface {
	List(ctx context.Context, userID uuid.UUID) ([]*pb.Account, error)
	Get(ctx context.Context, userID uuid.UUID, id int64) (*pb.Account, error)
	Create(ctx context.Context, params sqlc.CreateAccountParams, userSvc UserService) (*pb.Account, error)
	Update(ctx context.Context, params sqlc.UpdateAccountParams) error
	Delete(ctx context.Context, params sqlc.DeleteAccountParams) (int64, error)
	CheckUserAccountAccess(ctx context.Context, params sqlc.CheckUserAccountAccessParams) (bool, error)
}

type acctSvc struct {
	queries *sqlc.Queries
	log     *log.Logger
}

func (s *acctSvc) WithTx(tx pgx.Tx) AccountService {
	return &acctSvc{
		queries: s.queries.WithTx(tx),
		log:     s.log,
	}
}

func newAcctSvc(queries *sqlc.Queries, lg *log.Logger) AccountService {
	return &acctSvc{queries: queries, log: lg}
}

func (s *acctSvc) List(ctx context.Context, userID uuid.UUID) ([]*pb.Account, error) {
	rows, err := s.queries.ListAccounts(ctx, userID)
	if err != nil {
		return nil, wrapErr("AccountService.List", err)
	}

	accounts := make([]*pb.Account, len(rows))
	for i, row := range rows {
		accounts[i] = rowToPbAccount(
			row.ID, row.OwnerID, row.Name, row.Bank,
			pb.AccountType(row.AccountType), row.Alias,
			row.AnchorDate, row.AnchorBalanceCents, row.AnchorCurrency,
			row.MainCurrency, row.Colors,
			row.CreatedAt, row.UpdatedAt,
			row.BalanceCents, row.BalanceCurrency,
		)
	}
	return accounts, nil
}

func (s *acctSvc) Get(ctx context.Context, userID uuid.UUID, id int64) (*pb.Account, error) {
	row, err := s.queries.GetAccount(ctx, sqlc.GetAccountParams{
		UserID: userID,
		ID:     id,
	})

	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("AccountService.Get", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("AccountService.Get", err)
	}

	return rowToPbAccount(
		row.ID, row.OwnerID, row.Name, row.Bank,
		pb.AccountType(row.AccountType), row.Alias,
		row.AnchorDate, row.AnchorBalanceCents, row.AnchorCurrency,
		row.MainCurrency, row.Colors,
		row.CreatedAt, row.UpdatedAt,
		row.BalanceCents, row.BalanceCurrency,
	), nil
}

func (s *acctSvc) Create(ctx context.Context, params sqlc.CreateAccountParams, userSvc UserService) (*pb.Account, error) {
	// AnchorBalanceCents defaults to 0 if not provided, which is fine
	// Just ensure currency is set

	created, err := s.queries.CreateAccount(ctx, params)
	if err != nil {
		return nil, wrapErr("AccountService.Create", err)
	}

	if err := userSvc.EnsureDefaultAccount(ctx, params.OwnerID); err != nil {
		s.log.Warn("Failed to set default account for user", "user_id", params.OwnerID, "error", err)
	}

	// For newly created accounts, balance equals anchor balance
	return rowToPbAccount(
		created.ID, created.OwnerID, created.Name, created.Bank,
		pb.AccountType(created.AccountType), created.Alias,
		created.AnchorDate, created.AnchorBalanceCents, created.AnchorCurrency,
		created.MainCurrency, created.Colors,
		created.CreatedAt, created.UpdatedAt,
		created.AnchorBalanceCents, created.AnchorCurrency, // balance = anchor for new accounts
	), nil
}

func (s *acctSvc) Update(ctx context.Context, params sqlc.UpdateAccountParams) error {
	err := s.queries.UpdateAccount(ctx, params)
	if err != nil {
		return wrapErr("AccountService.Update", err)
	}

	// sync balances if anchor fields changed
	anchorFieldsChanged := params.AnchorDate != nil || params.AnchorBalanceCents != nil
	if anchorFieldsChanged {
		if err := s.queries.SyncAccountBalances(ctx, params.ID); err != nil {
			s.log.Warn("failed to sync account balances after updating anchor", "account_id", params.ID, "error", err)
		}
	}

	return nil
}

func (s *acctSvc) Delete(ctx context.Context, params sqlc.DeleteAccountParams) (int64, error) {
	affected, err := s.queries.DeleteAccount(ctx, params)
	if err != nil {
		return 0, wrapErr("AccountService.Delete", err)
	}
	return affected, nil
}

func (s *acctSvc) CheckUserAccountAccess(ctx context.Context, params sqlc.CheckUserAccountAccessParams) (bool, error) {
	access, err := s.queries.CheckUserAccountAccess(ctx, params)
	if err != nil {
		return false, wrapErr("AccountService.CheckUserAccountAccess", err)
	}
	return access, nil
}
