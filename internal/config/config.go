package config

import (
	"flag"
	"os"
	"time"
)

type Config struct {
	Port                 string
	InternalAPIKey       string
	LogLevel             string
	DatabaseURL          string
	ReceiptParserURL     string
	BetterAuthURL        string
	ReceiptParserTimeout time.Duration
}

func Load() Config {
	port := flag.String("port", "55555", "gRPC port")
	flag.Parse()

	internalAPIKey := os.Getenv("API_KEY")
	if internalAPIKey == "" {
		panic("API_KEY environment variable is required")
	}

	// TODO ping at startup to ensure BetterAuth is reachable
	betterAuthURL := os.Getenv("BETTER_AUTH_URL")
	if betterAuthURL == "" {
		panic("BETTER_AUTH_URL environment variable is required")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		panic("DATABASE_URL environment variable is required")
	}

	receiptParserURL := os.Getenv("ARIAN_RECEIPTS_URL")
	if receiptParserURL == "" {
		panic("ARIAN_RECEIPTS_URL environment variable is required")
	}

	timeoutStr := os.Getenv("RECEIPT_PARSER_TIMEOUT")
	if timeoutStr == "" {
		timeoutStr = "30s"
	}
	receiptParserTimeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		panic("invalid RECEIPT_PARSER_TIMEOUT value: must be a valid duration like '30s', '1m'")
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return Config{
		Port:                 ":" + *port,
		InternalAPIKey:       internalAPIKey,
		LogLevel:             logLevel,
		DatabaseURL:          databaseURL,
		ReceiptParserURL:     receiptParserURL,
		BetterAuthURL:        betterAuthURL,
		ReceiptParserTimeout: receiptParserTimeout,
	}
}
