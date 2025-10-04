package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
}

// NewClient creates a new webhook client
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Call makes an HTTP POST request to the webhook URL with the message
func (c *Client) Call(ctx context.Context, url string, message string) (string, error) {
	// Create request payload
	payload := WebhookRequest{
		Message: message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Try to parse as JSON first
	var webhookResp WebhookResponse
	if err := json.Unmarshal(body, &webhookResp); err == nil && webhookResp.Response != "" {
		return webhookResp.Response, nil
	}

	// If not JSON or no response field, return raw body
	return string(body), nil
}
