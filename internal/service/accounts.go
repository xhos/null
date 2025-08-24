package service

import (
	"ariand/internal/db/sqlc"
	"context"
	"database/sql"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"google.golang.org/genproto/googleapis/type/money"
)

type AccountService interface {
	ListForUser(ctx context.Context, userID uuid.UUID) ([]sqlc.ListAccountsForUserRow, error)
	GetForUser(ctx context.Context, userID uuid.UUID, id int64) (*sqlc.GetAccountForUserRow, error)
	Create(ctx context.Context, params sqlc.CreateAccountParams, userSvc UserService) (*sqlc.Account, error)
	Update(ctx context.Context, params sqlc.UpdateAccountParams) (*sqlc.Account, error)
	DeleteForUser(ctx context.Context, params sqlc.DeleteAccountForUserParams) (int64, error)
	GetUserAccountsCount(ctx context.Context, userID uuid.UUID) (int64, error)
	CheckUserAccountAccess(ctx context.Context, params sqlc.CheckUserAccountAccessParams) (bool, error)
	GetAnchorBalance(ctx context.Context, id int64) (*sqlc.GetAccountAnchorBalanceRow, error)
	GetBalance(ctx context.Context, accountID int64) (*money.Money, error)
	SetAnchor(ctx context.Context, params sqlc.SetAccountAnchorParams) error
	SyncBalances(ctx context.Context, accountID int64) error
}

type acctSvc struct {
	queries *sqlc.Queries
	log     *log.Logger
}

func normalizeAccountParams(params *sqlc.CreateAccountParams) {
	if params.AnchorCurrency == "" {
		params.AnchorCurrency = "CAD" // force everyone to be canadian, eh?
	}

	if params.AnchorBalance.IsZero() {
		params.AnchorBalance = decimal.Zero
	}
	// AnchorDate is set by the database
}

// WithTx creates a new service instance with a transaction
func (s *acctSvc) WithTx(tx pgx.Tx) AccountService {
	return &acctSvc{
		queries: s.queries.WithTx(tx),
		log:     s.log,
	}
}

func newAcctSvc(queries *sqlc.Queries, lg *log.Logger) AccountService {
	return &acctSvc{queries: queries, log: lg}
}

func (s *acctSvc) ListForUser(ctx context.Context, userID uuid.UUID) ([]sqlc.ListAccountsForUserRow, error) {
	accounts, err := s.queries.ListAccountsForUser(ctx, userID)
	if err != nil {
		return nil, wrapErr("AccountService.ListForUser", err)
	}
	return accounts, nil
}

func (s *acctSvc) GetForUser(ctx context.Context, userID uuid.UUID, id int64) (*sqlc.GetAccountForUserRow, error) {
	account, err := s.queries.GetAccountForUser(ctx, sqlc.GetAccountForUserParams{
		UserID: userID,
		ID:     id,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("AccountService.GetForUser", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("AccountService.GetForUser", err)
	}

	return &account, nil
}

func (s *acctSvc) Create(ctx context.Context, params sqlc.CreateAccountParams, userSvc UserService) (*sqlc.Account, error) {
	normalizeAccountParams(&params)

	created, err := s.queries.CreateAccount(ctx, params)
	if err != nil {
		return nil, wrapErr("AccountService.Create", err)
	}

	// ensure user has a default account set
	if err := userSvc.EnsureDefaultAccount(ctx, params.OwnerID); err != nil {
		s.log.Warn("Failed to set default account for user", "user_id", params.OwnerID, "error", err)
	}

	return &created, nil
}

func (s *acctSvc) Update(ctx context.Context, params sqlc.UpdateAccountParams) (*sqlc.Account, error) {
	updated, err := s.queries.UpdateAccount(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("AccountService.Update", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("AccountService.Update", err)
	}

	return &updated, nil
}

func (s *acctSvc) DeleteForUser(ctx context.Context, params sqlc.DeleteAccountForUserParams) (int64, error) {
	affected, err := s.queries.DeleteAccountForUser(ctx, params)
	if err != nil {
		return 0, wrapErr("AccountService.DeleteForUser", err)
	}
	return affected, nil
}

func (s *acctSvc) GetUserAccountsCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.queries.GetUserAccountsCount(ctx, userID)
	if err != nil {
		return 0, wrapErr("AccountService.GetUserAccountsCount", err)
	}
	return count, nil
}

func (s *acctSvc) CheckUserAccountAccess(ctx context.Context, params sqlc.CheckUserAccountAccessParams) (bool, error) {
	access, err := s.queries.CheckUserAccountAccess(ctx, params)
	if err != nil {
		return false, wrapErr("AccountService.CheckUserAccountAccess", err)
	}
	return access, nil
}

func (s *acctSvc) GetAnchorBalance(ctx context.Context, id int64) (*sqlc.GetAccountAnchorBalanceRow, error) {
	result, err := s.queries.GetAccountAnchorBalance(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("AccountService.GetAnchorBalance", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("AccountService.GetAnchorBalance", err)
	}

	return &result, nil
}

func (s *acctSvc) GetBalance(ctx context.Context, accountID int64) (*money.Money, error) {
	bal, err := s.queries.GetAccountBalance(ctx, accountID)
	if err != nil {
		return nil, wrapErr("AccountService.GetBalance", err)
	}
	if bal == nil {
		// Return zero money with default currency
		return &money.Money{
			CurrencyCode: "USD", // TODO: Get from account default currency
			Units:        0,
			Nanos:        0,
		}, nil
	}
	
	// Convert decimal to money
	balFloat, _ := bal.Float64()
	units := int64(balFloat)
	nanos := int32((balFloat - float64(units)) * 1e9)
	
	return &money.Money{
		CurrencyCode: "USD", // TODO: Get currency from account
		Units:        units,
		Nanos:        nanos,
	}, nil
}

func (s *acctSvc) SetAnchor(ctx context.Context, params sqlc.SetAccountAnchorParams) error {
	_, err := s.queries.SetAccountAnchor(ctx, params)
	if err != nil {
		return wrapErr("AccountService.SetAnchor", err)
	}
	return nil
}

func (s *acctSvc) SyncBalances(ctx context.Context, accountID int64) error {
	err := s.queries.SyncAccountBalances(ctx, accountID)
	if err != nil {
		return wrapErr("AccountService.SyncBalances", err)
	}
	return nil
}
