package gollm

// TODO: Temporarily commented out during MoneyWrapper migration
// This needs to be updated to work with the new JSONB money types

/*
import (
	"ariand/internal/ai"
	"ariand/internal/db/sqlc"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	gollm "github.com/teilomillet/gollm"
	llm "github.com/teilomillet/gollm/llm"
)
*/

/*
// ---------- plumbing for test stubs ----------

// llmNew is wrapped so unit-tests can replace it with a fake.
var llmNew = gollm.NewLLM

// generator is the part of gollm we actually need.
type generator interface {
	Generate(ctx context.Context, p *gollm.Prompt, opts ...llm.GenerateOption) (string, error)
}

// ---------- concrete provider ----------

type provider struct {
	llm  generator
	name string
}

func (p *provider) Name() string { return p.name }

// CategorizeTransaction fulfils ai.LLMProvider.
func (p *provider) CategorizeTransaction(
	ctx context.Context,
	tx sqlc.Transaction,
	allowedCategories []string,
) (string, float64, []string, error) {

	prompt := ai.BuildCategorizationPrompt(&tx, allowedCategories)
	raw, err := p.llm.Generate(ctx, gollm.NewPrompt(prompt))
	if err != nil {
		return "", 0, nil, fmt.Errorf("%s: %w", p.name, err)
	}

	res, err := ai.ParseCategorizationOutput(raw, allowedCategories)
	if err != nil {
		return "", 0, nil, err
	}
	return res.Category, res.Score, res.Suggestions, nil
}

// Chat creates a single-shot completion that follows the dialogue.
func (p *provider) Chat(ctx context.Context, history []ai.Message) (string, error) {
	var sb strings.Builder
	for _, m := range history {
		sb.WriteString(strings.Title(m.Role))
		sb.WriteString(": ")
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}
	sb.WriteString("Assistant:")

	resp, err := p.llm.Generate(ctx, gollm.NewPrompt(sb.String()))
	if err != nil {
		return "", fmt.Errorf("%s: %w", p.name, err)
	}
	return strings.TrimSpace(resp), nil
}

// Summarize is a thin helper; easy to refine later.
func (p *provider) Summarize(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(
		"Summarize the following text in a concise paragraph:\n\n%s\n\nSummary:",
		text,
	)
	resp, err := p.llm.Generate(ctx, gollm.NewPrompt(prompt))
	if err != nil {
		return "", fmt.Errorf("%s: %w", p.name, err)
	}
	return strings.TrimSpace(resp), nil
}

func (p *provider) ExtractMerchant(ctx context.Context, description string) (string, error) {
	prompt := ai.BuildMerchantExtractionPrompt(description)
	raw, err := p.llm.Generate(ctx, gollm.NewPrompt(prompt))
	if err != nil {
		return "", fmt.Errorf("%s extract merchant: %w", p.name, err)
	}
	// Clean the raw output
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, "Output:")
	clean = strings.TrimSpace(clean)
	unquoted, err := strconv.Unquote(clean)
	if err == nil {
		return unquoted, nil
	}

	// Fallback if Unquote fails (e.g., no quotes were present)
	return clean, nil
}

// ---------- builder helper ----------

// newBuilder returns an ai.Builder that initialises a GoLLM client
// for the specified engine ("openai", "anthropic", "ollama").
func newBuilder(engine string) ai.Builder {
	return func(model string, keys map[string]string) (ai.LLMProvider, error) {
		if model == "" {
			return nil, fmt.Errorf("%s builder: empty model", engine)
		}
		client, err := llmNew(
			gollm.SetProvider(engine),
			gollm.SetModel(model),
			gollm.SetAPIKey(keys[engine]),
			gollm.SetMaxTokens(768),
			gollm.SetOllamaEndpoint(os.Getenv("OLLAMA_ENDPOINT")),
		)
		if err != nil {
			return nil, fmt.Errorf("%s init: %w", engine, err)
		}
		return &provider{
			llm:  client,
			name: fmt.Sprintf("%s:%s", engine, model),
		}, nil
	}
}
*/
