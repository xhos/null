package api

import (
	"ariand/internal/db/sqlc"
	pb "ariand/internal/gen/arian/v1"
	"ariand/internal/rules"
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// TODO: validation can be simplified

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

	// validate that at least one action (category or merchant) is specified
	if req.Msg.CategoryId == nil && req.Msg.Merchant == nil {
		return nil, status.Error(codes.InvalidArgument, "At least one action (category_id or merchant) must be specified")
	}

	conditionsBytes, err := req.Msg.GetConditions().MarshalJSON()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid conditions JSON")
	}

	validationResult := rules.ValidateRuleJSONDetailed(conditionsBytes)
	if !validationResult.Valid {
		errorMsg := "Rule validation failed:"
		for _, validationErr := range validationResult.Errors {
			errorMsg += " " + validationErr.Error() + ";"
		}
		return nil, status.Error(codes.InvalidArgument, errorMsg)
	}

	normalizedRule, err := rules.NormalizeAndValidateRule(conditionsBytes)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Rule normalization failed: "+err.Error())
	}

	normalizedBytes, err := json.Marshal(normalizedRule)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to serialize normalized rule")
	}
	conditionsBytes = normalizedBytes

	params := sqlc.CreateRuleParams{
		UserID:     userID,
		RuleName:   req.Msg.GetRuleName(),
		Conditions: conditionsBytes,
	}

	if req.Msg.CategoryId != nil {
		params.CategoryID = *req.Msg.CategoryId
	}

	if req.Msg.Merchant != nil {
		params.Merchant = *req.Msg.Merchant
	}

	rule, err := s.services.Rules.Create(ctx, params)
	if err != nil {
		return nil, handleError(err)
	}

	// apply to existing transactions if requested
	if req.Msg.ApplyToExisting != nil && *req.Msg.ApplyToExisting {
		count, err := s.services.Rules.ApplyToExisting(ctx, userID, nil)
		if err != nil {
			// log but don't fail the request
			s.log.Warn("failed to apply rule to existing transactions", "rule_id", rule.RuleID, "error", err)
		} else {
			s.log.Info("applied rule to existing transactions", "rule_id", rule.RuleID, "count", count)
		}
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
	if req.Msg.Merchant != nil {
		params.Merchant = req.Msg.Merchant
	}
	if req.Msg.Conditions != nil {
		conditionsBytes, err := req.Msg.Conditions.MarshalJSON()
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "Invalid conditions JSON")
		}

		validationResult := rules.ValidateRuleJSONDetailed(conditionsBytes)
		if !validationResult.Valid {
			errorMsg := "Rule validation failed:"
			for _, validationErr := range validationResult.Errors {
				errorMsg += " " + validationErr.Error() + ";"
			}
			return nil, status.Error(codes.InvalidArgument, errorMsg)
		}

		normalizedRule, err := rules.NormalizeAndValidateRule(conditionsBytes)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "Rule normalization failed: "+err.Error())
		}

		normalizedBytes, err := json.Marshal(normalizedRule)
		if err != nil {
			return nil, status.Error(codes.Internal, "Failed to serialize normalized rule")
		}
		params.Conditions = normalizedBytes
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

	// apply to existing transactions if requested
	if req.Msg.ApplyToExisting != nil && *req.Msg.ApplyToExisting {
		count, err := s.services.Rules.ApplyToExisting(ctx, userID, nil)
		if err != nil {
			s.log.Warn("failed to apply rule to existing transactions", "rule_id", rule.RuleID, "error", err)
		} else {
			s.log.Info("applied rule to existing transactions", "rule_id", rule.RuleID, "count", count)
		}
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
		Merchant:      r.Merchant,
	}
}

func (s *Server) ValidateRule(ctx context.Context, req *connect.Request[pb.ValidateRuleRequest]) (*connect.Response[pb.ValidateRuleResponse], error) {
	conditionsBytes, err := req.Msg.GetConditions().MarshalJSON()
	if err != nil {
		return connect.NewResponse(&pb.ValidateRuleResponse{
			Valid: false,
			Errors: []*pb.ValidationError{{
				Field:   "conditions",
				Message: "Invalid JSON: " + err.Error(),
				Code:    "INVALID_JSON",
			}},
		}), nil
	}

	validationResult := rules.ValidateRuleJSONDetailed(conditionsBytes)

	response := &pb.ValidateRuleResponse{
		Valid:  validationResult.Valid,
		Errors: make([]*pb.ValidationError, len(validationResult.Errors)),
	}

	for i, validationErr := range validationResult.Errors {
		response.Errors[i] = &pb.ValidationError{
			Field:   validationErr.Field,
			Message: validationErr.Message,
			Code:    validationErr.Code,
		}
	}

	if validationResult.Valid {
		normalizedRule, err := rules.NormalizeAndValidateRule(conditionsBytes)
		if err == nil {
			normalizedBytes, err := json.Marshal(normalizedRule)
			if err == nil {
				var normalizedStruct structpb.Struct
				if err := normalizedStruct.UnmarshalJSON(normalizedBytes); err == nil {
					response.NormalizedConditions = &normalizedStruct
				}
			}
		}
	}

	return connect.NewResponse(response), nil
}
