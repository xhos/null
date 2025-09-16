package service

import (
	"ariand/internal/db/sqlc"
	"ariand/internal/types"
	"context"
	"database/sql"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/genproto/googleapis/type/money"
)

type AccountService interface {
	List(ctx context.Context, userID uuid.UUID) ([]sqlc.ListAccountsRow, error)
	Get(ctx context.Context, userID uuid.UUID, id int64) (*sqlc.GetAccountRow, error)
	Create(ctx context.Context, params sqlc.CreateAccountParams, userSvc UserService) (*sqlc.Account, error)
	Update(ctx context.Context, params sqlc.UpdateAccountParams) (*sqlc.Account, error)
	Delete(ctx context.Context, params sqlc.DeleteAccountParams) (int64, error)
	GetAccountCount(ctx context.Context, userID uuid.UUID) (int64, error)
	CheckUserAccountAccess(ctx context.Context, params sqlc.CheckUserAccountAccessParams) (bool, error)
	GetAnchorBalance(ctx context.Context, id int64) (*money.Money, error)
	GetBalance(ctx context.Context, accountID int64) (*money.Money, error)
	SetAnchor(ctx context.Context, params sqlc.SetAccountAnchorParams) error
	SyncBalances(ctx context.Context, accountID int64) error
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

func (s *acctSvc) List(ctx context.Context, userID uuid.UUID) ([]sqlc.ListAccountsRow, error) {
	accounts, err := s.queries.ListAccounts(ctx, userID)
	if err != nil {
		return nil, wrapErr("AccountService.List", err)
	}
	return accounts, nil
}

func (s *acctSvc) Get(ctx context.Context, userID uuid.UUID, id int64) (*sqlc.GetAccountRow, error) {
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
	needsDefaultBalance := len(params.AnchorBalance) == 0
	if needsDefaultBalance {
		defaultMoney := &money.Money{
			CurrencyCode: "CAD", // force everyone to be canadian, eh?
			Units:        0,
			Nanos:        0,
		}

		wrapper := types.WrapMoney(defaultMoney)
		jsonBytes, err := wrapper.Value()
		if err != nil {
			return nil, wrapErr("AccountService.Create", err)
		}

		if bytes, ok := jsonBytes.([]byte); ok {
			params.AnchorBalance = bytes
		}
	}

	created, err := s.queries.CreateAccount(ctx, params)
	if err != nil {
		return nil, wrapErr("AccountService.Create", err)
	}

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

func (s *acctSvc) Delete(ctx context.Context, params sqlc.DeleteAccountParams) (int64, error) {
	affected, err := s.queries.DeleteAccount(ctx, params)
	if err != nil {
		return 0, wrapErr("AccountService.Delete", err)
	}
	return affected, nil
}

func (s *acctSvc) GetAccountCount(ctx context.Context, userID uuid.UUID) (int64, error) {
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

func (s *acctSvc) GetAnchorBalance(ctx context.Context, id int64) (*money.Money, error) {
	result, err := s.queries.GetAccountAnchorBalance(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("AccountService.GetAnchorBalance", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("AccountService.GetAnchorBalance", err)
	}

	if result == nil {
		return &money.Money{
			CurrencyCode: "CAD",
			Units:        0,
			Nanos:        0,
		}, nil
	}

	return result.UnwrapMoney(), nil
}

func (s *acctSvc) GetBalance(ctx context.Context, accountID int64) (*money.Money, error) {
	anchorInfo, err := s.queries.GetAccountAnchorBalance(ctx, accountID)
	if err != nil {
		return nil, wrapErr("AccountService.GetBalance", err)
	}

	currentBalance, err := s.queries.GetAccountBalance(ctx, accountID)
	if err != nil {
		return nil, wrapErr("AccountService.GetBalance", err)
	}

	hasNoTransactions := currentBalance == nil
	if hasNoTransactions {
		anchorMoney := anchorInfo.UnwrapMoney()
		return &money.Money{
			CurrencyCode: anchorMoney.CurrencyCode,
			Units:        0,
			Nanos:        0,
		}, nil
	}

	return currentBalance.UnwrapMoney(), nil
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
