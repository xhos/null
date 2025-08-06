package service

import (
	"ariand/internal/ai"
	"ariand/internal/config"
	"ariand/internal/db"
	"ariand/internal/receiptparser"
	"ariand/internal/storage"
	"context"
	"time"

	"github.com/charmbracelet/log"
)

type Services struct {
	Transactions TransactionService
	Categories   CategoryService
	Accounts     AccountService
	Dashboard    DashboardService
	Receipts     ReceiptService
	Users        UserService
	Auth         AuthService
}

func New(database *db.DB, lg *log.Logger, cfg *config.Config, aiMgr *ai.Manager) (*Services, error) {
	queries := database.Queries
	catSvc := newCatSvc(queries, lg.WithPrefix("cat"))

	// initialize the gRPC receipt parser client
	parserClient, err := receiptparser.New(cfg.ReceiptParserURL, cfg.ReceiptParserTimeout)
	if err != nil {
		return nil, err
	}

	// test connection to receipt parser service
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := parserClient.TestConnection(ctx); err != nil {
		lg.Warn("receipt parser service is not reachable - receipt parsing will not be available",
			"url", cfg.ReceiptParserURL,
			"error", err.Error())
	} else {
		lg.Info("receipt parser service connected successfully", "url", cfg.ReceiptParserURL)
	}

	return &Services{
		Transactions: newTxnSvc(queries, lg.WithPrefix("txn"), catSvc, aiMgr),
		Categories:   catSvc,
		Accounts:     newAcctSvc(queries, lg.WithPrefix("acct")),
		Dashboard:    newDashSvc(queries),
		Users:        newUserSvc(queries, database, lg.WithPrefix("user")), //TODO: WHY PASS DB?
		Auth:         newAuthSvc(queries, lg.WithPrefix("auth")),
		Receipts:     newReceiptSvc(queries, parserClient, storage.NewLocalStorage("/tmp/receipts", "/api/receipts/images"), lg.WithPrefix("receipt")),
	}, nil
}
