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

func New(database *db.DB, lg *log.Logger, cfg *config.Config) (*Services, error) {
	queries := database.Queries
	catSvc := newCatSvc(queries, lg.WithPrefix("cat"))

	ruleSvc := newCatRuleSvc(queries, lg.WithPrefix("rules"))

	exchangeClient := exchange.NewClient(cfg.ExchangeAPIURL)

	return &Services{
		Transactions: newTxnSvc(queries, lg.WithPrefix("txn"), catSvc, ruleSvc, exchangeClient),
		Categories:   catSvc,
		Rules:        ruleSvc,
		Accounts:     newAcctSvc(queries, lg.WithPrefix("acct")),
		Dashboard:    newDashSvc(queries),
		Users:        newUserSvc(queries, lg.WithPrefix("user")),
		Backup:       newBackupSvc(queries),
	}, nil
}
