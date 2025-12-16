package service

import (
	"ariand/internal/db/sqlc"
	"context"
	"database/sql"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/genproto/googleapis/type/money"
)

// centsToMoney converts cents to google.type.Money
func centsToMoney(cents int64, currency string) *money.Money {
	return &money.Money{
		CurrencyCode: currency,
		Units:        cents / 100,
		Nanos:        int32((cents % 100) * 10_000_000),
	}
}

type AccountService interface {
	List(ctx context.Context, userID uuid.UUID) ([]sqlc.Account, error)
	Get(ctx context.Context, userID uuid.UUID, id int64) (*sqlc.Account, error)
	Create(ctx context.Context, params sqlc.CreateAccountParams, userSvc UserService) (*sqlc.Account, error)
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

func (s *acctSvc) List(ctx context.Context, userID uuid.UUID) ([]sqlc.Account, error) {
	accounts, err := s.queries.ListAccounts(ctx, userID)
	if err != nil {
		return nil, wrapErr("AccountService.List", err)
	}
	return accounts, nil
}

func (s *acctSvc) Get(ctx context.Context, userID uuid.UUID, id int64) (*sqlc.Account, error) {
	account, err := s.queries.GetAccount(ctx, sqlc.GetAccountParams{
		UserID: userID,
		ID:     id,
	})

	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("AccountService.Get", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("AccountService.Get", err)
	}

	return &account, nil
}

func (s *acctSvc) Create(ctx context.Context, params sqlc.CreateAccountParams, userSvc UserService) (*sqlc.Account, error) {
	// AnchorBalanceCents defaults to 0 if not provided, which is fine
	// Just ensure currency is set

	created, err := s.queries.CreateAccount(ctx, params)
	if err != nil {
		return nil, wrapErr("AccountService.Create", err)
	}

	if err := userSvc.EnsureDefaultAccount(ctx, params.OwnerID); err != nil {
		s.log.Warn("Failed to set default account for user", "user_id", params.OwnerID, "error", err)
	}

	return &created, nil
}

func (s *acctSvc) Update(ctx context.Context, params sqlc.UpdateAccountParams) error {
	err := s.queries.UpdateAccount(ctx, params)
	if err != nil {
		return wrapErr("AccountService.Update", err)
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
