package api

import (
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
)

func (s *Server) ListRules(ctx context.Context, req *connect.Request[pb.ListRulesRequest]) (*connect.Response[pb.ListRulesResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	rules, err := s.services.Rules.List(ctx, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.ListRulesResponse{
		Rules: mapSlice(rules, toProtoRule),
	}), nil
}

func (s *Server) GetRule(ctx context.Context, req *connect.Request[pb.GetRuleRequest]) (*connect.Response[pb.GetRuleResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	ruleID, err := parseUUID(req.Msg.GetRuleId())
	if err != nil {
		return nil, err
	}

	rule, err := s.services.Rules.Get(ctx, userID, ruleID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.GetRuleResponse{
		Rule: toProtoRule(rule),
	}), nil
}

func (s *Server) CreateRule(ctx context.Context, req *connect.Request[pb.CreateRuleRequest]) (*connect.Response[pb.CreateRuleResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	categoryID := req.Msg.GetCategoryId()

	conditionsBytes, err := req.Msg.GetConditions().MarshalJSON()
	if err != nil {
		return nil, handleError(err)
	}

	params := sqlc.CreateRuleParams{
		UserID:     userID,
		RuleName:   req.Msg.GetRuleName(),
		CategoryID: categoryID,
		Conditions: conditionsBytes,
	}

	rule, err := s.services.Rules.Create(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.CreateRuleResponse{
		Rule: toProtoRule(rule),
	}), nil
}

func (s *Server) UpdateRule(ctx context.Context, req *connect.Request[pb.UpdateRuleRequest]) (*connect.Response[pb.UpdateRuleResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	ruleID, err := parseUUID(req.Msg.GetRuleId())
	if err != nil {
		return nil, err
	}

	params := sqlc.UpdateRuleParams{
		RuleID: ruleID,
		UserID: userID,
	}

	if req.Msg.RuleName != nil {
		params.RuleName = req.Msg.RuleName
	}
	if req.Msg.CategoryId != nil {
		params.CategoryID = req.Msg.CategoryId
	}
	if req.Msg.Conditions != nil {
		conditionsBytes, err := req.Msg.Conditions.MarshalJSON()
		if err != nil {
			return nil, handleError(err)
		}
		params.Conditions = conditionsBytes
	}
	if req.Msg.IsActive != nil {
		params.IsActive = req.Msg.IsActive
	}
	if req.Msg.PriorityOrder != nil {
		priority := int32(*req.Msg.PriorityOrder)
		params.PriorityOrder = &priority
	}

	rule, err := s.services.Rules.Update(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.UpdateRuleResponse{
		Rule: toProtoRule(rule),
	}), nil
}

func (s *Server) DeleteRule(ctx context.Context, req *connect.Request[pb.DeleteRuleRequest]) (*connect.Response[pb.DeleteRuleResponse], error) {
	userID, err := getUserID(ctx)
	if err != nil {
		return nil, err
	}

	ruleID, err := parseUUID(req.Msg.GetRuleId())
	if err != nil {
		return nil, err
	}

	affected, err := s.services.Rules.Delete(ctx, userID, ruleID)
	if err != nil {
		return nil, handleError(err)
	}

	return connect.NewResponse(&pb.DeleteRuleResponse{
		AffectedRows: affected,
	}), nil
}

func toProtoRule(r *sqlc.TransactionRule) *pb.Rule {
	if r == nil {
		return nil
	}

	var conditions *structpb.Struct
	if len(r.Conditions) > 0 {
		conditions = &structpb.Struct{}
		if err := conditions.UnmarshalJSON(r.Conditions); err == nil {
			// conditions successfully unmarshaled
		} else {
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

	return &pb.Rule{
		RuleId:        r.RuleID.String(),
		UserId:        r.UserID.String(),
		RuleName:      r.RuleName,
		CategoryId:    r.CategoryID,
		Conditions:    conditions,
		IsActive:      isActive,
		PriorityOrder: r.PriorityOrder,
		RuleSource:    r.RuleSource,
		CreatedAt:     toProtoTimestamp(&r.CreatedAt),
		UpdatedAt:     toProtoTimestamp(&r.UpdatedAt),
		LastAppliedAt: toProtoTimestamp(r.LastAppliedAt),
		TimesApplied:  timesApplied,
	}
}
