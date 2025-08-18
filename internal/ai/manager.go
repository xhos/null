package ai

import (
	"fmt"
	"os"
	"sync"
)

// Builder constructs a concrete LLMProvider for a given model name.
// The keys map supplies per-vendor API keys (openai, anthropic, …).
type Builder func(model string, keys map[string]string) (LLMProvider, error)

// Manager stores provider builders and hands out ready-to-use LLMProvider
// instances on demand.  It is safe for concurrent use.
type Manager struct {
	mu       sync.RWMutex
	builders map[string]Builder
}

// defaultManager is the singleton used by the rest of the application.
var defaultManager = NewManager()

// NewManager returns an empty registry – handy for isolated tests.
func NewManager() *Manager {
	return &Manager{
		builders: make(map[string]Builder),
	}
}

// GetManager exposes the singleton for init() hooks in sub-packages.
func GetManager() *Manager { return defaultManager }

// Register adds/overwrites the builder under the given name.
func (m *Manager) Register(name string, b Builder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.builders[name] = b
}

// GetProvider instantiates the requested provider/model or returns an error.
func (m *Manager) GetProvider(providerName, model string) (LLMProvider, error) {
	m.mu.RLock()
	builder, ok := m.builders[providerName]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown AI provider %q", providerName)
	}

	keys := map[string]string{
		"openai":    os.Getenv("OPENAI_API_KEY"),
		"anthropic": os.Getenv("ANTHROPIC_API_KEY"),
		"ollama":    os.Getenv("OLLAMA_API_KEY"),
		"gemini":    os.Getenv("GOOGLE_API_KEY"), // placeholder for future use
	}

	return builder(model, keys)
}
