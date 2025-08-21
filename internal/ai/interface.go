package ai

import (
	"ariand/internal/db/sqlc"
	"context"
)

// LLMProvider defines the minimal, provider-agnostic contract
// every large-language-model backend must satisfy.
//
// Only CategorizeTransaction and Chat are needed right now.
// Summarize is included to keep the interface future-proof.
type LLMProvider interface {
	// Name returns a human-readable identifier
	// such as "openai:gpt-4o" or "ollama:phi3".
	Name() string

	// CategorizeTransaction analyses a single banking
	// transaction and returns:
	//   – category slug            (e.g. "food.groceries")
	//   – confidence in [0,1]
	//   – optional alternative suggestions
	CategorizeTransaction(ctx context.Context, tx sqlc.Transaction, allowedCategories []string) (category string, confidence float64, suggestions []string, err error)

	// Chat generates the assistant’s next reply given the
	// complete message history.
	Chat(ctx context.Context, history []Message) (response string, err error)

	// Summarize condenses an arbitrary block of text.
	// Providers may return an ErrNotImplemented until
	// the application actually needs summarisation.
	Summarize(ctx context.Context, text string) (summary string, err error)

	// ExtractMerchant analyzes a transaction description to identify
	// and clean up the merchant's name.
	// e.g., "strbck coffee" -> "Starbucks"
	ExtractMerchant(ctx context.Context, description string) (merchant string, err error)
}
