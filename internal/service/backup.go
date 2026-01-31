package service

import (
	"context"
	"encoding/json"
	"time"

	"null/internal/backup"
	"null/internal/db/sqlc"
	pb "null/internal/gen/null/v1"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ----- interface ---------------------------------------------------------------------------

type BackupService interface {
	ExportAll(ctx context.Context, userID uuid.UUID) (*backup.Backup, error)
	ImportAll(ctx context.Context, userID uuid.UUID, data *backup.Backup) error
	BackupToProto(b *backup.Backup) *pb.Backup
	BackupFromProto(pb *pb.Backup) *backup.Backup
}

type backupSvc struct {
	db *sqlc.Queries
}

func newBackupSvc(db *sqlc.Queries) BackupService {
	return &backupSvc{db: db}
}

// ----- methods -----------------------------------------------------------------------------

func (s *backupSvc) ExportAll(ctx context.Context, userID uuid.UUID) (*backup.Backup, error) {
	return backup.ExportAll(ctx, s.db, userID)
}

func (s *backupSvc) ImportAll(ctx context.Context, userID uuid.UUID, data *backup.Backup) error {
	return backup.ImportAll(ctx, s.db, userID, data)
}

// ----- conversion helpers ------------------------------------------------------------------

func (s *backupSvc) BackupToProto(b *backup.Backup) *pb.Backup {
	if b == nil {
		return nil
	}

	protoCategories := make([]*pb.CategoryData, len(b.Categories))
	for i, cat := range b.Categories {
		protoCategories[i] = &pb.CategoryData{
			Slug:  cat.Slug,
			Color: cat.Color,
		}
	}

	protoAccounts := make([]*pb.AccountData, len(b.Accounts))
	for i, acc := range b.Accounts {
		protoAccounts[i] = &pb.AccountData{
			Name:          acc.Name,
			Bank:          acc.Bank,
			AccountType:   acc.AccountType,
			Alias:         acc.Alias,
			AnchorDate:    toProtoTimestamp(acc.AnchorDate),
			AnchorBalance: acc.AnchorBalance,
			MainCurrency:  acc.MainCurrency,
			Colors:        acc.Colors,
		}
	}

	protoTransactions := make([]*pb.TransactionData, len(b.Transactions))
	for i, tx := range b.Transactions {
		protoTransactions[i] = &pb.TransactionData{
			AccountName:   tx.AccountName,
			TxDate:        timestamppb.New(tx.TxDate),
			TxAmount:      tx.TxAmount,
			TxDirection:   tx.TxDirection,
			TxDesc:        tx.TxDesc,
			BalanceAfter:  tx.BalanceAfter,
			Merchant:      tx.Merchant,
			CategorySlug:  tx.CategorySlug,
			UserNotes:     tx.UserNotes,
			ForeignAmount: tx.ForeignAmount,
			ExchangeRate:  tx.ExchangeRate,
		}
	}

	protoRules := make([]*pb.RuleData, len(b.Rules))
	for i, rule := range b.Rules {
		conditionsJSON, _ := json.Marshal(rule.Conditions)
		protoRules[i] = &pb.RuleData{
			RuleName:       rule.RuleName,
			CategorySlug:   rule.CategorySlug,
			Merchant:       rule.Merchant,
			ConditionsJson: string(conditionsJSON),
			IsActive:       rule.IsActive,
			PriorityOrder:  rule.PriorityOrder,
			RuleSource:     rule.RuleSource,
		}
	}

	return &pb.Backup{
		Version:      b.Version,
		ExportedAt:   timestamppb.New(b.ExportedAt),
		Categories:   protoCategories,
		Accounts:     protoAccounts,
		Transactions: protoTransactions,
		Rules:        protoRules,
	}
}

func (s *backupSvc) BackupFromProto(pbBackup *pb.Backup) *backup.Backup {
	if pbBackup == nil {
		return nil
	}

	categories := make([]backup.CategoryData, len(pbBackup.Categories))
	for i, cat := range pbBackup.Categories {
		categories[i] = backup.CategoryData{
			Slug:  cat.Slug,
			Color: cat.Color,
		}
	}

	accounts := make([]backup.AccountData, len(pbBackup.Accounts))
	for i, acc := range pbBackup.Accounts {
		var anchorDate *time.Time
		if acc.AnchorDate != nil {
			t := acc.AnchorDate.AsTime()
			anchorDate = &t
		}

		accounts[i] = backup.AccountData{
			Name:          acc.Name,
			Bank:          acc.Bank,
			AccountType:   acc.AccountType,
			Alias:         acc.Alias,
			AnchorDate:    anchorDate,
			AnchorBalance: acc.AnchorBalance,
			MainCurrency:  acc.MainCurrency,
			Colors:        acc.Colors,
		}
	}

	transactions := make([]backup.TransactionData, len(pbBackup.Transactions))
	for i, tx := range pbBackup.Transactions {
		transactions[i] = backup.TransactionData{
			AccountName:   tx.AccountName,
			TxDate:        tx.TxDate.AsTime(),
			TxAmount:      tx.TxAmount,
			TxDirection:   tx.TxDirection,
			TxDesc:        tx.TxDesc,
			BalanceAfter:  tx.BalanceAfter,
			Merchant:      tx.Merchant,
			CategorySlug:  tx.CategorySlug,
			UserNotes:     tx.UserNotes,
			ForeignAmount: tx.ForeignAmount,
			ExchangeRate:  tx.ExchangeRate,
		}
	}

	rules := make([]backup.RuleData, len(pbBackup.Rules))
	for i, rule := range pbBackup.Rules {
		var conditions map[string]interface{}
		json.Unmarshal([]byte(rule.ConditionsJson), &conditions)

		rules[i] = backup.RuleData{
			RuleName:      rule.RuleName,
			CategorySlug:  rule.CategorySlug,
			Merchant:      rule.Merchant,
			Conditions:    conditions,
			IsActive:      rule.IsActive,
			PriorityOrder: rule.PriorityOrder,
			RuleSource:    rule.RuleSource,
		}
	}

	return &backup.Backup{
		Version:      pbBackup.Version,
		ExportedAt:   pbBackup.ExportedAt.AsTime(),
		Categories:   categories,
		Accounts:     accounts,
		Transactions: transactions,
		Rules:        rules,
	}
}
