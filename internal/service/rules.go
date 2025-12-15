package service

import (
	"ariand/internal/db/sqlc"
	"ariand/internal/rules"
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

	ApplyToTransaction(ctx context.Context, userID uuid.UUID, tx *sqlc.Transaction, account *sqlc.Account) (*RuleMatchResult, error)
	ApplyToExisting(ctx context.Context, userID uuid.UUID, transactionIDs []int64) (int, error)
}

type RuleMatchResult struct {
	CategoryID *int64
	Merchant   *string
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
	err := s.queries.UpdateRule(ctx, params)
	if err != nil {
		return nil, wrapErr("RuleService.Update", err)
	}

	// Fetch and return updated rule
	rule, err := s.queries.GetRule(ctx, sqlc.GetRuleParams{
		UserID: params.UserID,
		RuleID: params.RuleID,
	})
	if err != nil {
		return nil, wrapErr("RuleService.Update.Get", err)
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

func (s *catRuleSvc) ApplyToTransaction(ctx context.Context, userID uuid.UUID, tx *sqlc.Transaction, account *sqlc.Account) (*RuleMatchResult, error) {
	s.log.Info("ApplyToTransaction called",
		"user_id", userID,
		"tx_desc", tx.TxDesc,
		"merchant", tx.Merchant)

	activeRules, err := s.queries.GetActiveRules(ctx, userID)
	if err != nil {
		return nil, wrapErr("RuleService.ApplyToTransaction", err)
	}

	s.log.Info("Fetched active rules", "count", len(activeRules))

	result := &RuleMatchResult{}
	matchedCount := 0

	for _, rule := range activeRules {
		s.log.Debug("Evaluating rule",
			"rule_id", rule.RuleID,
			"rule_name", rule.RuleName,
			"conditions", string(rule.Conditions))

		conditions, err := rules.ParseRuleConditions(rule.Conditions)
		if err != nil {
			s.log.Warn("failed to parse rule conditions", "rule_id", rule.RuleID, "error", err)
			continue
		}

		matches, err := rules.EvaluateRule(conditions, tx, account)
		if err != nil {
			s.log.Warn("failed to evaluate rule", "rule_id", rule.RuleID, "error", err)
			continue
		}

		if !matches {
			s.log.Debug("Rule did not match", "rule_id", rule.RuleID)
			continue
		}

		matchedCount++
		s.log.Info("Rule matched!",
			"rule_id", rule.RuleID,
			"rule_name", rule.RuleName,
			"category_id", rule.CategoryID,
			"merchant", rule.Merchant)

		// first matching rule for category wins
		if result.CategoryID == nil && rule.CategoryID != nil {
			result.CategoryID = rule.CategoryID
			s.log.Info("Setting category from rule", "category_id", *rule.CategoryID)
		}

		// first matching rule for merchant wins
		if result.Merchant == nil && rule.Merchant != nil {
			result.Merchant = rule.Merchant
			s.log.Info("Setting merchant from rule", "merchant", *rule.Merchant)
		}

		// stop if we have both
		if result.CategoryID != nil && result.Merchant != nil {
			s.log.Info("Both fields matched, stopping evaluation")
			break
		}
	}

	s.log.Info("Rule application completed",
		"total_rules", len(activeRules),
		"matched_rules", matchedCount,
		"final_category_id", result.CategoryID,
		"final_merchant", result.Merchant)

	return result, nil
}

func (s *catRuleSvc) ApplyToExisting(ctx context.Context, userID uuid.UUID, transactionIDs []int64) (int, error) {
	// fetch transactions that aren't manually set
	includeManuallySet := false
	transactions, err := s.queries.GetTransactionsForRuleApplication(ctx, sqlc.GetTransactionsForRuleApplicationParams{
		UserID:             userID,
		TransactionIds:     transactionIDs,
		IncludeManuallySet: &includeManuallySet,
	})
	if err != nil {
		return 0, wrapErr("RuleService.ApplyToExisting.FetchTransactions", err)
	}

	if len(transactions) == 0 {
		return 0, nil
	}

	// load active rules once
	activeRules, err := s.queries.GetActiveRules(ctx, userID)
	if err != nil {
		return 0, wrapErr("RuleService.ApplyToExisting.FetchRules", err)
	}

	// group transactions by matching rules for bulk updates
	type updateKey struct {
		categoryID int64
		merchant   string
	}
	updateGroups := make(map[updateKey][]int64)

	for _, tx := range transactions {
		// fetch account for rule evaluation
		account, err := s.queries.GetAccount(ctx, sqlc.GetAccountParams{
			UserID: userID,
			ID:     tx.AccountID,
		})
		if err != nil {
			s.log.Warn("failed to fetch account for rule application", "account_id", tx.AccountID, "error", err)
			continue
		}

		ruleResult := s.evaluateRulesForTransaction(activeRules, &tx, &account)

		// skip if no rules matched
		if ruleResult.CategoryID == nil && ruleResult.Merchant == nil {
			continue
		}

		// create grouping key
		key := updateKey{
			categoryID: 0,
			merchant:   "",
		}
		if ruleResult.CategoryID != nil {
			key.categoryID = *ruleResult.CategoryID
		}
		if ruleResult.Merchant != nil {
			key.merchant = *ruleResult.Merchant
		}

		updateGroups[key] = append(updateGroups[key], tx.ID)
	}

	// bulk update each group
	totalUpdated := 0
	for key, txIDs := range updateGroups {
		categoryID := int64(0)
		merchant := ""

		if key.categoryID > 0 {
			categoryID = key.categoryID
		}
		if key.merchant != "" {
			merchant = key.merchant
		}

		affected, err := s.queries.BulkApplyRuleToTransactions(ctx, sqlc.BulkApplyRuleToTransactionsParams{
			CategoryID:     categoryID,
			Merchant:       merchant,
			TransactionIds: txIDs,
			UserID:         userID,
		})
		if err != nil {
			s.log.Warn("failed to bulk apply rules", "error", err)
			continue
		}

		totalUpdated += int(affected)
	}

	return totalUpdated, nil
}

// evaluateRulesForTransaction is the same logic as ApplyToTransaction but without ctx/queries
func (s *catRuleSvc) evaluateRulesForTransaction(activeRules []sqlc.TransactionRule, tx *sqlc.Transaction, account *sqlc.Account) *RuleMatchResult {
	result := &RuleMatchResult{}

	for _, rule := range activeRules {
		conditions, err := rules.ParseRuleConditions(rule.Conditions)
		if err != nil {
			continue
		}

		matches, err := rules.EvaluateRule(conditions, tx, account)
		if err != nil || !matches {
			continue
		}

		if result.CategoryID == nil && rule.CategoryID != nil {
			result.CategoryID = rule.CategoryID
		}

		if result.Merchant == nil && rule.Merchant != nil {
			result.Merchant = rule.Merchant
		}

		if result.CategoryID != nil && result.Merchant != nil {
			break
		}
	}

	return result
}
