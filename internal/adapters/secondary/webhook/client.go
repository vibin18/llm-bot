package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// Client implements WebhookClient interface
type Client struct {
	httpClient *http.Client
	timeout    time.Duration
}

// WebhookRequest represents the payload sent to webhook
type WebhookRequest struct {
	Message string `json:"message"`
}

// WebhookResponse represents the response from webhook
type WebhookResponse struct {
	Response string `json:"response"`
	Output   string `json:"output"` // Support for "output" field
}

// NewClient creates a new webhook client
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			// No timeout here - we'll use context timeout instead for per-request control
			Timeout: 0,
		},
		timeout: timeout,
	}
}

// Call makes an HTTP POST request to the webhook URL with the message
func (c *Client) Call(ctx context.Context, url string, message string) (*domain.WebhookResponse, error) {
	// Create request payload
	payload := WebhookRequest{
		Message: message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with context (allows timeout override)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Get content type from response header
	contentType := resp.Header.Get("Content-Type")

	// Determine response type based on Content-Type header
	result := &domain.WebhookResponse{
		ContentType: contentType,
		Content:     body,
	}

	// Handle different content types
	if strings.HasPrefix(contentType, "image/jpeg") || strings.HasPrefix(contentType, "image/jpg") {
		result.ContentType = "image/jpeg"
	} else if strings.HasPrefix(contentType, "image/png") {
		result.ContentType = "image/png"
	} else {
		// Default to text - try to parse as JSON first
		var webhookResp WebhookResponse
		if err := json.Unmarshal(body, &webhookResp); err == nil {
			// Check for "output" field first, then "response" field
			if webhookResp.Output != "" {
				result.ContentType = "text"
				result.TextContent = webhookResp.Output
			} else if webhookResp.Response != "" {
				result.ContentType = "text"
				result.TextContent = webhookResp.Response
			} else {
				// JSON but no recognized fields, use raw body
				result.ContentType = "text"
				result.TextContent = string(body)
			}
		} else {
			// If not JSON, treat as plain text
			result.ContentType = "text"
			result.TextContent = string(body)
		}
	}

	return result, nil
}
