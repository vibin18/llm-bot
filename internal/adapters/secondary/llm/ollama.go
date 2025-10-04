package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// OllamaProvider implements LLMProvider interface
type OllamaProvider struct {
	llm         *ollama.LLM
	model       string
	temperature float64
	timeout     time.Duration
}

// NewOllamaProvider creates a new Ollama LLM provider
func NewOllamaProvider(url, model string, temperature float64, timeout time.Duration) (*OllamaProvider, error) {
	llm, err := ollama.New(
		ollama.WithServerURL(url),
		ollama.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	return &OllamaProvider{
		llm:         llm,
		model:       model,
		temperature: temperature,
		timeout:     timeout,
	}, nil
}

// Generate generates a response from the LLM
func (p *OllamaProvider) Generate(ctx context.Context, request *domain.LLMRequest) (*domain.LLMResponse, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Build prompt with context
	prompt := p.buildPrompt(request)

	// Generate response
	response, err := llms.GenerateFromSinglePrompt(
		ctx,
		p.llm,
		prompt,
		llms.WithTemperature(p.temperature),
		llms.WithModel(p.model),
	)

	if err != nil {
		return &domain.LLMResponse{
			Error: fmt.Errorf("failed to generate response: %w", err),
		}, err
	}

	return &domain.LLMResponse{
		Content: response,
		Error:   nil,
	}, nil
}

// IsAvailable checks if the LLM service is available
func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
	// Try a simple generation to check availability
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := llms.GenerateFromSinglePrompt(
		ctx,
		p.llm,
		"test",
		llms.WithModel(p.model),
	)

	return err == nil
}

// buildPrompt constructs a prompt with conversation context
func (p *OllamaProvider) buildPrompt(request *domain.LLMRequest) string {
	var builder strings.Builder

	// Add system instruction
	builder.WriteString("You are a helpful AI assistant in a WhatsApp group chat. ")
	builder.WriteString("Provide concise, friendly, and helpful responses. ")
	builder.WriteString("Keep your answers brief and to the point.\n\n")

	// Add conversation context if available
	if len(request.Context) > 0 {
		builder.WriteString("Recent conversation:\n")
		// Only include last 5 messages for context
		start := 0
		if len(request.Context) > 5 {
			start = len(request.Context) - 5
		}

		for i := start; i < len(request.Context); i++ {
			msg := request.Context[i]
			if msg.IsFromBot {
				builder.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
			} else {
				builder.WriteString(fmt.Sprintf("User: %s\n", msg.Content))
			}
		}
		builder.WriteString("\n")
	}

	// Add current prompt
	builder.WriteString(fmt.Sprintf("User: %s\nAssistant:", request.Prompt))

	return builder.String()
}
