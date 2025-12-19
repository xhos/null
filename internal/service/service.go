package service

import (
	"ariand/internal/config"
	"ariand/internal/db"
	"ariand/internal/exchange"

	"github.com/charmbracelet/log"
)

type Services struct {
	Transactions TransactionService
	Categories   CategoryService
	Rules        RuleService
	Accounts     AccountService
	Dashboard    DashboardService
	Users        UserService
	Backup       BackupService
}

func New(database *db.DB, logger *log.Logger, cfg *config.Config) (*Services, error) {
	queries := database.Queries
	catSvc := newCatSvc(queries, logger.WithPrefix("cat"))
	ruleSvc := newCatRuleSvc(queries, logger.WithPrefix("rules"))
	exchangeClient := exchange.NewClient(cfg.ExchangeAPIURL)

	return &Services{
		Transactions: newTxnSvc(queries, logger.WithPrefix("txn"), catSvc, ruleSvc, exchangeClient),
		Categories:   catSvc,
		Rules:        ruleSvc,
		Accounts:     newAcctSvc(queries, logger.WithPrefix("acct")),
		Dashboard:    newDashSvc(queries),
		Users:        newUserSvc(queries, logger.WithPrefix("user")),
		Backup:       newBackupSvc(queries),
	}, nil
}
