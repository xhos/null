package ai

// TODO: Temporarily commented out during MoneyWrapper migration
// This needs to be updated to work with the new JSONB money types

/*
import (
	"ariand/internal/db/sqlc"
	"ariand/internal/types"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// moneyToFloat converts MoneyWrapper to float64
func moneyToFloat(wrapper *types.MoneyWrapper) (float64, string) {
	if wrapper == nil || wrapper.Money == nil {
		return 0.0, "USD"
	}
	amount := float64(wrapper.Money.Units) + float64(wrapper.Money.Nanos)/1e9
	return amount, wrapper.Money.CurrencyCode
}

// BuildCategorizationPrompt constructs a best-practice prompt:
//   - Enumerates allowed categories for the "category" field.
//   - Uses few-shot examples with fenced JSON.
//   - Specifies that "suggestions" must be NEW slugs (not in allowedCategories),
//     to consider adding to the taxonomy.
//   - Always responds with exactly one JSON object.
func BuildCategorizationPrompt(tx *sqlc.Transaction, allowedCategories []string) string {
	val := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	
	amount, currency := moneyToFloat(tx.TxAmount)
	
	return fmt.Sprintf(
		`You are a financial assistant. Categorize this transaction.

Respond with exactly one JSON object (no extra text), matching this schema:
{
  "category":    string,       # one of: %s
  "score":       number,       # confidence 0.0–1.0
  "suggestions": array[string] # 3-5 NEW category slugs (not in the allowed list)
}

Allowed categories:
  %s

Examples:

Input:
  Merchant:    "CafeBistro"
  Description: "Lunch sandwich"
  Amount:      12.75 USD
Output:
{"category":"food.takeout","score":0.92,"suggestions":[]}

Input:
  Merchant:    "SkyAir"
  Description: "International flight fare"
  Amount:      450.00 USD
Output:
{"category":"other","score":0.85,"suggestions":["transport.airfare","transport.flight"]}

Now categorize:
  Merchant:    %q
  Description: %q
  Amount:      %.2f %s
  Date:        %s
`,
		strings.Join(allowedCategories, ", "),
		strings.Join(allowedCategories, ", "),
		val(tx.Merchant),
		val(tx.TxDesc),
		amount, currency,
		tx.TxDate.Format("2006-01-02T15:04:05Z07:00"),
	)
}
*/

/*
// CategoryResult holds the parsed response from ParseCategorizationOutput.
type CategoryResult struct {
	Category    string   `json:"category"`
	Score       float64  `json:"score"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// ParseCategorizationOutput strips fences, extracts JSON, and unmarshals.
// It clamps the score to [0,1]. If the model did not provide suggestions,
// it injects a fallback list of other allowed categories to ensure the
// field is never empty, but real new-category suggestions come from the model.
func ParseCategorizationOutput(raw string, allowedCategories []string) (CategoryResult, error) {
	var res CategoryResult

	clean := stripFences(strings.TrimSpace(raw))
	jsonText := extractJSON(clean)

	if err := json.Unmarshal([]byte(jsonText), &res); err != nil {
		return res, fmt.Errorf("malformed JSON: %w (resp=%q)", err, raw)
	}
	if res.Category == "" {
		return res, fmt.Errorf("empty category from LLM")
	}

	// Clamp confidence to [0,1]
	res.Score = clamp(res.Score)

	// Enforce category membership; if invalid, treat as "other"
	if !isAllowedCategory(res.Category, allowedCategories) {
		res.Category = "other"
	}

	return res, nil
}

// isAllowedCategory checks membership in AllowedCategories.
func isAllowedCategory(cat string, allowedCategories []string) bool {
	for _, a := range allowedCategories {
		if a == cat {
			return true
		}
	}
	return false
}

// clamp confines x to [0,1].
func clamp(x float64) float64 {
	switch {
	case x < 0:
		return 0
	case x > 1:
		return 1
	default:
		return x
	}
}

// stripFences removes leading/trailing ``` fences.
func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
	}

	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "```")

	return strings.TrimSpace(s)
}

// extractJSON returns the first balanced {...} block.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return s[start:]
}

// BuildMerchantExtractionPrompt creates a prompt to extract a clean merchant name.
func BuildMerchantExtractionPrompt(description string) string {
	return fmt.Sprintf(
		`Your task is to identify the merchant name from a raw transaction description.
        Return only the cleaned, proper name of the merchant and nothing else.

        Examples:
        - Input: "AMZ Mktp US" -> Output: "Amazon"
        - Input: "SQ *SQ *THE COFFEE SHOP" -> Output: "The Coffee Shop"
        - Input: "strbck coffee #1234" -> Output: "Starbucks"
        - Input: "UBER   EATS" -> Output: "Uber Eats"

        Now, identify the merchant from this description:
        Input: %q -> Output:`,
		description,
	)
}
*/
