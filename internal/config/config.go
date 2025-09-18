package config

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type Config struct {
	Address string // the address on which the server listens
	APIKey  string // internal API key for authenticating requests

	DatabaseURL   string // database connection URL
	BetterAuthURL string // better-auth service URL

	ReceiptParserURL     string        // receipt parser service URL
	ReceiptParserTimeout time.Duration // timeout for receipt parser requests

	ExchangeAPIURL string // exchange rate API URL

	LogLevel log.Level // logging level
}

// parseAddress ensures the address is in the correct format for network listeners.
// If the input is just a port (e.g. "55555"), it returns ":55555".
// If the input is already an address (e.g. "0.0.0.0:55555" or ":55555"), it returns it unchanged.
// Examples:
//
//	parseAddress("55555")         // ":55555"
//	parseAddress(":55555")        // ":55555"
//	parseAddress("0.0.0.0:55555") // "0.0.0.0:55555"
func parseAddress(port string) string {
	port = strings.TrimSpace(port)
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}

// Load reads configuration from environment variables and command-line flags
func Load() Config {
	address := flag.String("port", "55555", "listen address or port (e.g. 55555, :55555, 0.0.0.0:55555)")

	flag.Parse()

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
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

	exchangeAPIURL := os.Getenv("EXCHANGE_API_URL")
	if exchangeAPIURL == "" {
		panic("EXCHANGE_API_URL environment variable is required")
	}

	timeoutStr := os.Getenv("RECEIPT_PARSER_TIMEOUT")
	if timeoutStr == "" {
		timeoutStr = "30s"
	}
	receiptParserTimeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		panic("invalid RECEIPT_PARSER_TIMEOUT value: must be a valid duration like '30s', '1m'")
	}

	logLevel, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = log.InfoLevel
	}

	return Config{
		Address:              parseAddress(*address),
		APIKey:               apiKey,
		DatabaseURL:          databaseURL,
		BetterAuthURL:        betterAuthURL,
		ReceiptParserURL:     receiptParserURL,
		ReceiptParserTimeout: receiptParserTimeout,
		ExchangeAPIURL:       exchangeAPIURL,
		LogLevel:             logLevel,
	}
}
