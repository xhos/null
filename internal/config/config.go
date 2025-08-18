package config

import (
	"flag"
	"os"
	"time"
)

type Config struct {
	Port                 string
	APIKey               string
	LogLevel             string
	DatabaseURL          string
	ReceiptParserURL     string
	ReceiptParserTimeout time.Duration
}

func Load() Config {
	port := flag.String("port", "55555", "gRPC port")
	flag.Parse()

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		panic("API_KEY environment variable is required")
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
		panic("RECEIPT_PARSER_TIMEOUT environment variable is required")
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
		APIKey:               apiKey,
		LogLevel:             logLevel,
		DatabaseURL:          databaseURL,
		ReceiptParserURL:     receiptParserURL,
		ReceiptParserTimeout: receiptParserTimeout,
	}
}
