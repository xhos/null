package service

import (
	"ariand/internal/db/sqlc"
	"context"
	"database/sql"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

type RuleService interface {
	List(ctx context.Context, userID uuid.UUID) ([]sqlc.TransactionRule, error)
	Get(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID) (*sqlc.TransactionRule, error)
	Create(ctx context.Context, params sqlc.CreateRuleParams) (*sqlc.TransactionRule, error)
	Update(ctx context.Context, params sqlc.UpdateRuleParams) (*sqlc.TransactionRule, error)
	Delete(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID) (int64, error)
}

type catRuleSvc struct {
	queries *sqlc.Queries
	log     *log.Logger
}

func newCatRuleSvc(queries *sqlc.Queries, lg *log.Logger) RuleService {
	return &catRuleSvc{queries: queries, log: lg}
}

func (s *catRuleSvc) List(ctx context.Context, userID uuid.UUID) ([]sqlc.TransactionRule, error) {
	rules, err := s.queries.ListRules(ctx, userID)
	if err != nil {
		return nil, wrapErr("RuleService.List", err)
	}
	return rules, nil
}

func (s *catRuleSvc) Get(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID) (*sqlc.TransactionRule, error) {
	rule, err := s.queries.GetRule(ctx, sqlc.GetRuleParams{
		RuleID: ruleID,
		UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("RuleService.Get", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("RuleService.Get", err)
	}
	return &rule, nil
}

func (s *catRuleSvc) Create(ctx context.Context, params sqlc.CreateRuleParams) (*sqlc.TransactionRule, error) {
	rule, err := s.queries.CreateRule(ctx, params)
	if err != nil {
		return nil, wrapErr("RuleService.Create", err)
	}
	return &rule, nil
}

func (s *catRuleSvc) Update(ctx context.Context, params sqlc.UpdateRuleParams) (*sqlc.TransactionRule, error) {
	rule, err := s.queries.UpdateRule(ctx, params)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wrapErr("RuleService.Update", ErrNotFound)
	}
	if err != nil {
		return nil, wrapErr("RuleService.Update", err)
	}
	return &rule, nil
}

func (s *catRuleSvc) Delete(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID) (int64, error) {
	affected, err := s.queries.DeleteRule(ctx, sqlc.DeleteRuleParams{
		RuleID: ruleID,
		UserID: userID,
	})
	if err != nil {
		return 0, wrapErr("RuleService.Delete", err)
	}
	return affected, nil
}
