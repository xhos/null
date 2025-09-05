package config

import (
	"flag"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port                 string // actually "listen addr"
	InternalAPIKey       string
	LogLevel             string
	DatabaseURL          string
	ReceiptParserURL     string
	BetterAuthURL        string
	ReceiptParserTimeout time.Duration
}

func normalizeListen(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ":55555"
	}
	// if the user supplied a host:port (contains ':'), trust it as-is
	if strings.Contains(v, ":") {
		return v
	}
	// bare port → prepend ':'
	return ":" + v
}

func Load() Config {
	portFlag := flag.String("port", "55555", "listen address or port (e.g. 55555, :55555, 0.0.0.0:55555)")
	flag.Parse()

	internalAPIKey := os.Getenv("API_KEY")
	if internalAPIKey == "" {
		panic("API_KEY environment variable is required")
	}

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
		Port:                 normalizeListen(*portFlag),
		InternalAPIKey:       internalAPIKey,
		LogLevel:             logLevel,
		DatabaseURL:          databaseURL,
		ReceiptParserURL:     receiptParserURL,
		BetterAuthURL:        betterAuthURL,
		ReceiptParserTimeout: receiptParserTimeout,
	}
}
