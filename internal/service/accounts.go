package service

import (
	sqlc "ariand/internal/db/sqlc"
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
	GetForUser(ctx context.Context, params sqlc.GetAccountForUserParams) (*sqlc.GetAccountForUserRow, error)
	Create(ctx context.Context, params sqlc.CreateAccountParams, userSvc UserService) (*sqlc.Account, error)
	Update(ctx context.Context, params sqlc.UpdateAccountParams) (*sqlc.Account, error)
	DeleteForUser(ctx context.Context, params sqlc.DeleteAccountForUserParams) error
	GetUserAccountsCount(ctx context.Context, userID uuid.UUID) (int64, error)
	CheckUserAccountAccess(ctx context.Context, params sqlc.CheckUserAccountAccessParams) (bool, error)
	GetAnchorBalance(ctx context.Context, id int64) (*sqlc.GetAccountAnchorBalanceRow, error)
	GetBalance(ctx context.Context, accountID int64) (*money.Money, error)
	SetAnchor(ctx context.Context, params sqlc.SetAccountAnchorParams) error
	SyncBalances(ctx context.Context, accountID int64) error
	AddCollaborator(ctx context.Context, params sqlc.AddAccountCollaboratorParams) (*sqlc.AccountUser, error)
	RemoveCollaborator(ctx context.Context, params sqlc.RemoveAccountCollaboratorParams) error
	ListCollaborators(ctx context.Context, params sqlc.ListAccountCollaboratorsParams) ([]sqlc.ListAccountCollaboratorsRow, error)
	GetCollaboratorCount(ctx context.Context, accountID int64) (int64, error)
	CheckCollaborator(ctx context.Context, params sqlc.CheckAccountCollaboratorParams) (bool, error)
	TransferOwnership(ctx context.Context, params sqlc.TransferAccountOwnershipParams) error
	LeaveCollaboration(ctx context.Context, params sqlc.LeaveAccountCollaborationParams) error
	ListUserCollaborations(ctx context.Context, userID uuid.UUID) ([]sqlc.ListUserCollaborationsRow, error)
	RemoveUserFromAllAccounts(ctx context.Context, userID uuid.UUID) error
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

func (s *acctSvc) GetForUser(ctx context.Context, params sqlc.GetAccountForUserParams) (*sqlc.GetAccountForUserRow, error) {
	account, err := s.queries.GetAccountForUser(ctx, params)
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

func (s *acctSvc) DeleteForUser(ctx context.Context, params sqlc.DeleteAccountForUserParams) error {
	_, err := s.queries.DeleteAccountForUser(ctx, params)
	if err != nil {
		return wrapErr("AccountService.DeleteForUser", err)
	}
	return nil
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
	return bal, nil
}

func (s *acctSvc) AddCollaborator(ctx context.Context, params sqlc.AddAccountCollaboratorParams) (*sqlc.AccountUser, error) {
	collaborator, err := s.queries.AddAccountCollaborator(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, wrapErr("AccountService.AddCollaborator", err)
	}

	return &collaborator, nil
}

func (s *acctSvc) RemoveCollaborator(ctx context.Context, params sqlc.RemoveAccountCollaboratorParams) error {
	_, err := s.queries.RemoveAccountCollaborator(ctx, params)
	if err != nil {
		return wrapErr("AccountService.RemoveCollaborator", err)
	}
	return nil
}

func (s *acctSvc) ListCollaborators(ctx context.Context, params sqlc.ListAccountCollaboratorsParams) ([]sqlc.ListAccountCollaboratorsRow, error) {
	collaborators, err := s.queries.ListAccountCollaborators(ctx, params)
	if err != nil {
		return nil, wrapErr("AccountService.ListCollaborators", err)
	}
	return collaborators, nil
}

func (s *acctSvc) GetCollaboratorCount(ctx context.Context, accountID int64) (int64, error) {
	count, err := s.queries.GetAccountCollaboratorCount(ctx, accountID)
	if err != nil {
		return 0, wrapErr("AccountService.GetCollaboratorCount", err)
	}
	return count, nil
}

func (s *acctSvc) LeaveCollaboration(ctx context.Context, params sqlc.LeaveAccountCollaborationParams) error {
	_, err := s.queries.LeaveAccountCollaboration(ctx, params)
	if err != nil {
		return wrapErr("AccountService.LeaveCollaboration", err)
	}
	return nil
}

func (s *acctSvc) ListUserCollaborations(ctx context.Context, userID uuid.UUID) ([]sqlc.ListUserCollaborationsRow, error) {
	collaborations, err := s.queries.ListUserCollaborations(ctx, userID)
	if err != nil {
		return nil, wrapErr("AccountService.ListUserCollaborations", err)
	}
	return collaborations, nil
}

func (s *acctSvc) RemoveUserFromAllAccounts(ctx context.Context, userID uuid.UUID) error {
	_, err := s.queries.RemoveUserFromAllAccounts(ctx, userID)
	if err != nil {
		return wrapErr("AccountService.RemoveUserFromAllAccounts", err)
	}
	return nil
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

func (s *acctSvc) CheckCollaborator(ctx context.Context, params sqlc.CheckAccountCollaboratorParams) (bool, error) {
	isCollaborator, err := s.queries.CheckAccountCollaborator(ctx, params)
	if err != nil {
		return false, wrapErr("AccountService.CheckCollaborator", err)
	}
	return isCollaborator, nil
}

func (s *acctSvc) TransferOwnership(ctx context.Context, params sqlc.TransferAccountOwnershipParams) error {
	_, err := s.queries.TransferAccountOwnership(ctx, params)
	if err != nil {
		return wrapErr("AccountService.TransferOwnership", err)
	}
	return nil
}
