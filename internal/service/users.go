package service

import (
	"ariand/internal/db"
	"ariand/internal/db/sqlc"
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

type UserService interface {
	Get(ctx context.Context, id uuid.UUID) (*sqlc.User, error)
	GetByEmail(ctx context.Context, email string) (*sqlc.User, error)
	Create(ctx context.Context, params sqlc.CreateUserParams) (*sqlc.User, error)
	Update(ctx context.Context, params sqlc.UpdateUserParams) (*sqlc.User, error)
	UpdateDisplayName(ctx context.Context, params sqlc.UpdateUserDisplayNameParams) (*sqlc.User, error)
	SetDefaultAccount(ctx context.Context, params sqlc.SetUserDefaultAccountParams) (*sqlc.User, error)
	EnsureDefaultAccount(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]sqlc.User, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type userSvc struct {
	queries *sqlc.Queries
	db      *db.DB
	log     *log.Logger
}

func newUserSvc(queries *sqlc.Queries, database *db.DB, lg *log.Logger) UserService {
	return &userSvc{queries: queries, db: database, log: lg}
}

func (s *userSvc) Get(ctx context.Context, id uuid.UUID) (*sqlc.User, error) {
	user, err := s.queries.GetUser(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("UserService.Get", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("UserService.Get", err)
	}
	return &user, nil
}

func (s *userSvc) GetByEmail(ctx context.Context, email string) (*sqlc.User, error) {
	user, err := s.queries.GetUserByEmail(ctx, strings.ToLower(email))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("UserService.GetByEmail", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("UserService.GetByEmail", err)
	}

	return &user, nil
}

func (s *userSvc) Create(ctx context.Context, params sqlc.CreateUserParams) (*sqlc.User, error) {
	params.Email = strings.ToLower(params.Email)

	user, err := s.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, wrapErr("UserService.Create", err)
	}

	return &user, nil
}

func (s *userSvc) Update(ctx context.Context, params sqlc.UpdateUserParams) (*sqlc.User, error) {
	if params.Email != nil {
		normalized := strings.ToLower(*params.Email)
		params.Email = &normalized
	}

	user, err := s.queries.UpdateUser(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("UserService.Update", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("UserService.Update", err)
	}

	return &user, nil
}

func (s *userSvc) UpdateDisplayName(ctx context.Context, params sqlc.UpdateUserDisplayNameParams) (*sqlc.User, error) {
	user, err := s.queries.UpdateUserDisplayName(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("UserService.UpdateDisplayName", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("UserService.UpdateDisplayName", err)
	}

	return &user, nil
}

// TODO: Ensure deleting user that is a collab on some accounts doesnt delete the account
func (s *userSvc) Delete(ctx context.Context, id uuid.UUID) error {

	// use transaction for cascade delete
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return wrapErr("UserService.Delete", err)
	}
	defer tx.Rollback(ctx)

	txQueries := s.queries.WithTx(tx)

	// remove from all accounts
	_, err = txQueries.RemoveUserFromAllAccounts(ctx, id)
	if err != nil {
		return wrapErr("UserService.Delete.RemoveFromAccounts", err)
	}

	// delete user
	_, err = txQueries.DeleteUser(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return wrapErr("UserService.Delete", ErrNotFound)
	}

	if err != nil {
		return wrapErr("UserService.Delete", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return wrapErr("UserService.Delete.Commit", err)
	}

	s.log.Info("User deleted with cascade", "user_id", id)

	return nil
}

func (s *userSvc) List(ctx context.Context) ([]sqlc.User, error) {
	users, err := s.queries.ListUsers(ctx)
	if err != nil {
		return nil, wrapErr("UserService.List", err)
	}
	return users, nil
}

func (s *userSvc) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	exists, err := s.queries.CheckUserExists(ctx, id)
	if err != nil {
		return false, wrapErr("UserService.Exists", err)
	}
	return exists, nil
}

func (s *userSvc) SetDefaultAccount(ctx context.Context, params sqlc.SetUserDefaultAccountParams) (*sqlc.User, error) {
	user, err := s.queries.SetUserDefaultAccount(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("UserService.SetDefaultAccount", ErrNotFound)
	}

	if err != nil {
		return nil, wrapErr("UserService.SetDefaultAccount", err)
	}

	return &user, nil
}

func (s *userSvc) EnsureDefaultAccount(ctx context.Context, userID uuid.UUID) error {
	user, err := s.queries.GetUser(ctx, userID)
	if err != nil {
		return wrapErr("UserService.EnsureDefaultAccount", err)
	}

	if user.DefaultAccountID != nil {
		return nil
	}

	firstAccountID, err := s.queries.GetUserFirstAccount(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return wrapErr("UserService.EnsureDefaultAccount", err)
	}

	_, err = s.queries.SetUserDefaultAccount(ctx, sqlc.SetUserDefaultAccountParams{
		ID:               userID,
		DefaultAccountID: firstAccountID,
	})
	if err != nil {
		return wrapErr("UserService.EnsureDefaultAccount", err)
	}

	s.log.Info("Set default account for user", "user_id", userID, "account_id", firstAccountID)
	return nil
}
