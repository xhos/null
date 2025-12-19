package service

import (
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"ariand/internal/rules"
	"context"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ----- interface ---------------------------------------------------------------------------

type RuleService interface {
	Create(ctx context.Context, userID uuid.UUID, ruleName string, conditions []byte, categoryID *int64, merchant *string) (*pb.Rule, error)
	Get(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID) (*pb.Rule, error)
	Update(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID, ruleName *string, conditions []byte, categoryID *int64, merchant *string) error
	Delete(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID) (int64, error)
	List(ctx context.Context, userID uuid.UUID) ([]*pb.Rule, error)

	ApplyToTransaction(ctx context.Context, userID uuid.UUID, tx *sqlc.Transaction, account *sqlc.GetAccountRow) (*RuleMatchResult, error)
	ApplyToExisting(ctx context.Context, userID uuid.UUID, transactionIDs []int64) (int, error)
}

type catRuleSvc struct {
	queries *sqlc.Queries
	log     *log.Logger
}

func newCatRuleSvc(queries *sqlc.Queries, logger *log.Logger) RuleService {
	return &catRuleSvc{queries: queries, log: logger}
}

// ----- methods -----------------------------------------------------------------------------

type RuleMatchResult struct {
	CategoryID *int64
	Merchant   *string
}

func (s *catRuleSvc) Create(ctx context.Context, userID uuid.UUID, ruleName string, conditions []byte, categoryID *int64, merchant *string) (*pb.Rule, error) {
	params := sqlc.CreateRuleParams{
		UserID:     userID,
		RuleName:   ruleName,
		Conditions: conditions,
	}
	if categoryID != nil {
		params.CategoryID = *categoryID
	}
	if merchant != nil {
		params.Merchant = *merchant
	}

	rule, err := s.queries.CreateRule(ctx, params)
	if err != nil {
		return nil, wrapErr("RuleService.Create", err)
	}

	return ruleToPb(&rule), nil
}

func (s *catRuleSvc) Get(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID) (*pb.Rule, error) {
	rule, err := s.queries.GetRule(ctx, sqlc.GetRuleParams{
		RuleID: ruleID,
		UserID: userID,
	})
	if err != nil {
		return nil, wrapErr("RuleService.Get", err)
	}

	return ruleToPb(&rule), nil
}

func (s *catRuleSvc) Update(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID, ruleName *string, conditions []byte, categoryID *int64, merchant *string) error {
	params := sqlc.UpdateRuleParams{
		RuleID:   ruleID,
		UserID:   userID,
		RuleName: ruleName,
	}

	if len(conditions) > 0 {
		params.Conditions = conditions
	}
	if categoryID != nil {
		params.CategoryID = categoryID
	}
	if merchant != nil {
		params.Merchant = merchant
	}

	err := s.queries.UpdateRule(ctx, params)
	if err != nil {
		return wrapErr("RuleService.Update", err)
	}

	return nil
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

func (s *catRuleSvc) List(ctx context.Context, userID uuid.UUID) ([]*pb.Rule, error) {
	rows, err := s.queries.ListRules(ctx, userID)
	if err != nil {
		return nil, wrapErr("RuleService.List", err)
	}

	result := make([]*pb.Rule, len(rows))
	for i := range rows {
		result[i] = ruleToPb(&rows[i])
	}

	return result, nil
}

func (s *catRuleSvc) ApplyToTransaction(ctx context.Context, userID uuid.UUID, tx *sqlc.Transaction, account *sqlc.GetAccountRow) (*RuleMatchResult, error) {
	activeRules, err := s.queries.GetActiveRules(ctx, userID)
	if err != nil {
		return nil, wrapErr("RuleService.ApplyToTransaction", err)
	}

	return s.evaluateRulesForTransaction(activeRules, tx, account), nil
}

func (s *catRuleSvc) ApplyToExisting(ctx context.Context, userID uuid.UUID, transactionIDs []int64) (int, error) {
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

	activeRules, err := s.queries.GetActiveRules(ctx, userID)
	if err != nil {
		return 0, wrapErr("RuleService.ApplyToExisting.FetchRules", err)
	}

	type updateKey struct {
		categoryID int64
		merchant   string
	}

	updateGroups := make(map[updateKey][]int64)

	for _, tx := range transactions {
		account, err := s.queries.GetAccount(ctx, sqlc.GetAccountParams{
			UserID: userID,
			ID:     tx.AccountID,
		})
		if err != nil {
			s.log.Warn("failed to fetch account for rule application", "account_id", tx.AccountID, "error", err)
			continue
		}

		ruleResult := s.evaluateRulesForTransaction(activeRules, &tx, &account)

		noMatch := ruleResult.CategoryID == nil && ruleResult.Merchant == nil
		if noMatch {
			continue
		}

		key := updateKey{}
		if ruleResult.CategoryID != nil {
			key.categoryID = *ruleResult.CategoryID
		}
		if ruleResult.Merchant != nil {
			key.merchant = *ruleResult.Merchant
		}

		updateGroups[key] = append(updateGroups[key], tx.ID)
	}

	totalUpdated := 0
	for key, txIDs := range updateGroups {
		affected, err := s.queries.BulkApplyRuleToTransactions(ctx, sqlc.BulkApplyRuleToTransactionsParams{
			CategoryID:     key.categoryID,
			Merchant:       key.merchant,
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

// ----- conversion helpers ------------------------------------------------------------------

func ruleToPb(r *sqlc.TransactionRule) *pb.Rule {
	var conditions *structpb.Struct
	if len(r.Conditions) > 0 {
		conditions = &structpb.Struct{}
		if err := conditions.UnmarshalJSON(r.Conditions); err != nil {
			conditions = nil
		}
	}

	isActive := false
	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	timesApplied := int32(0)
	if r.TimesApplied != nil {
		timesApplied = *r.TimesApplied
	}

	rule := &pb.Rule{
		RuleId:        r.RuleID.String(),
		UserId:        r.UserID.String(),
		RuleName:      r.RuleName,
		CategoryId:    r.CategoryID,
		Merchant:      r.Merchant,
		Conditions:    conditions,
		IsActive:      isActive,
		PriorityOrder: r.PriorityOrder,
		RuleSource:    r.RuleSource,
		TimesApplied:  timesApplied,
	}

	if !r.CreatedAt.IsZero() {
		rule.CreatedAt = timestamppb.New(r.CreatedAt)
	}
	if !r.UpdatedAt.IsZero() {
		rule.UpdatedAt = timestamppb.New(r.UpdatedAt)
	}
	if r.LastAppliedAt != nil {
		rule.LastAppliedAt = timestamppb.New(*r.LastAppliedAt)
	}

	return rule
}

// ----- internal helpers --------------------------------------------------------------------

func (s *catRuleSvc) evaluateRulesForTransaction(activeRules []sqlc.TransactionRule, tx *sqlc.Transaction, account *sqlc.GetAccountRow) *RuleMatchResult {
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
