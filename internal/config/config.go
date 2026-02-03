package config

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

type Config struct {
	ListenAddress string
	APIKey        string // for for internal service communication

	DatabaseURL     string
	NullWebURL      string
	NullReceiptsURL string
	ExchangeAPIURL  string

	LogLevel  log.Level
	LogFormat string // "json" | "text"
}

// safely parse whatever port or address the user provides
// handdles cases like "8080", ":8080", "127.0.0.1:8080"
func parseAddress(port string) string {
	port = strings.TrimSpace(port)
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}

func Load() Config {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		panic("API_KEY environment variable is required")
	}

	nullWebURL := os.Getenv("NULL_WEB_URL")
	if nullWebURL == "" {
		panic("NULL_WEB_URL environment variable is required")
	}

	nullReceiptsURL := os.Getenv("NULL_RECEIPTS_URL")
	if nullReceiptsURL == "" {
		// TODO: need to make this log print uniform with the app-wide logger
		// configuration. perhaps create the logger here, use it and then return it?
		log.Warn("NULL_RECEIPTS_URL is not set!")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		panic("DATABASE_URL environment variable is required")
	}

	exchangeAPIURL := os.Getenv("EXCHANGE_API_URL")
	if exchangeAPIURL == "" {
		panic("EXCHANGE_API_URL environment variable is required")
	}

	logLevel, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = log.InfoLevel
	}

	logFormat := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	if logFormat != "json" && logFormat != "text" {
		logFormat = "text"
	}

	listenAddr := os.Getenv("LISTEN_ADDRESS")
	if listenAddr == "" {
		listenAddr = "127.0.0.1:55555"
	}

	return Config{
		ListenAddress:   parseAddress(listenAddr),
		APIKey:          apiKey,
		DatabaseURL:     databaseURL,
		NullWebURL:      nullWebURL,
		NullReceiptsURL: nullReceiptsURL,
		ExchangeAPIURL:  exchangeAPIURL,
		LogLevel:        logLevel,
		LogFormat:       logFormat,
	}
}
