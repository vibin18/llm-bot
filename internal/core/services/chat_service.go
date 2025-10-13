package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// ChatService handles chat message processing
type ChatService struct {
	llmProvider    domain.LLMProvider
	repository     domain.MessageRepository
	whatsapp       domain.WhatsAppClient
	groupMgr       domain.GroupManager
	webhookClient  domain.WebhookClient
	triggerWords   []string
	webhookConfigs []domain.WebhookConfig
	configMu       sync.RWMutex
	logger         *slog.Logger
}

// NewChatService creates a new chat service
func NewChatService(
	llmProvider domain.LLMProvider,
	repository domain.MessageRepository,
	whatsapp domain.WhatsAppClient,
	groupMgr domain.GroupManager,
	webhookClient domain.WebhookClient,
	triggerWords []string,
	webhookConfigs []domain.WebhookConfig,
	logger *slog.Logger,
) *ChatService {
	return &ChatService{
		llmProvider:    llmProvider,
		repository:     repository,
		whatsapp:       whatsapp,
		groupMgr:       groupMgr,
		webhookClient:  webhookClient,
		triggerWords:   triggerWords,
		webhookConfigs: webhookConfigs,
		logger:         logger,
	}
}

// ProcessMessage processes an incoming message
func (s *ChatService) ProcessMessage(ctx context.Context, message *domain.Message) error {
	// Validate group is allowed
	if !s.groupMgr.IsAllowed(message.GroupJID) {
		s.logger.Debug("Message from non-allowed group", "group", message.GroupJID)
		return nil
	}

	// Check if message starts with any trigger word OR is a reply to bot
	s.configMu.RLock()
	triggerWords := s.triggerWords
	s.configMu.RUnlock()

	if len(triggerWords) > 0 && !message.IsReplyToBot {
		trimmedContent := strings.TrimSpace(message.Content)
		triggered := false
		var matchedTrigger string

		for _, trigger := range triggerWords {
			if strings.HasPrefix(trimmedContent, trigger) {
				triggered = true
				matchedTrigger = trigger
				// Remove trigger word from message content
				message.Content = strings.TrimSpace(strings.TrimPrefix(message.Content, trigger))
				break
			}
		}

		if !triggered {
			s.logger.Debug("Message doesn't start with any trigger word and is not a reply",
				"triggers", triggerWords,
				"content", message.Content)
			return nil
		}

		s.logger.Debug("Message triggered", "trigger", matchedTrigger)

		// Check for webhook sub-trigger
		if webhook := s.findMatchingWebhook(message.Content); webhook != nil {
			return s.processWebhookMessage(ctx, message, webhook)
		}
	} else if message.IsReplyToBot {
		s.logger.Debug("Message is a reply to bot", "content", message.Content)
	}

	// Save incoming message
	if err := s.repository.Save(ctx, message); err != nil {
		s.logger.Error("Failed to save message", "error", err)
		return fmt.Errorf("failed to save message: %w", err)
	}

	s.logger.Info("Processing message",
		"group", message.GroupJID,
		"sender", message.Sender,
		"content", message.Content)

	// Get conversation context
	context, err := s.repository.GetByGroupJID(ctx, message.GroupJID, 10)
	if err != nil {
		s.logger.Error("Failed to get context", "error", err)
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Generate LLM response - convert []*domain.Message to []domain.Message
	contextMsgs := make([]domain.Message, len(context))
	for i, msg := range context {
		contextMsgs[i] = *msg
	}

	llmRequest := &domain.LLMRequest{
		Prompt:  message.Content,
		Context: contextMsgs,
	}

	response, err := s.llmProvider.Generate(ctx, llmRequest)
	if err != nil || response.Error != nil {
		s.logger.Error("Failed to generate LLM response", "error", err)

		// Send user-friendly error message as a reply
		errorMsg := "Sorry, I cannot process this request right now due to a technical error. Please try again later."
		if err := s.whatsapp.SendReply(ctx, message.GroupJID, errorMsg, message.ID, message.Sender); err != nil {
			s.logger.Error("Failed to send error message", "error", err)
		}
		return fmt.Errorf("failed to generate response: %w", err)
	}

	s.logger.Info("Generated response", "content", response.Content)

	// Send response back to WhatsApp as a reply to the original message
	if err := s.whatsapp.SendReply(ctx, message.GroupJID, response.Content, message.ID, message.Sender); err != nil {
		s.logger.Error("Failed to send message", "error", err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Save bot response
	botMessage := &domain.Message{
		ID:        fmt.Sprintf("bot-%d", message.Timestamp.Unix()),
		GroupJID:  message.GroupJID,
		Sender:    "bot",
		Content:   response.Content,
		Timestamp: message.Timestamp,
		IsFromBot: true,
	}

	if err := s.repository.Save(ctx, botMessage); err != nil {
		s.logger.Error("Failed to save bot message", "error", err)
	}

	return nil
}

// findMatchingWebhook finds a webhook config that matches the message content
func (s *ChatService) findMatchingWebhook(content string) *domain.WebhookConfig {
	trimmedContent := strings.TrimSpace(content)

	s.configMu.RLock()
	defer s.configMu.RUnlock()

	for i := range s.webhookConfigs {
		webhook := &s.webhookConfigs[i]
		if strings.HasPrefix(trimmedContent, webhook.SubTrigger) {
			return webhook
		}
	}

	return nil
}

// processWebhookMessage handles messages that trigger a webhook
func (s *ChatService) processWebhookMessage(ctx context.Context, message *domain.Message, webhook *domain.WebhookConfig) error {
	// Remove sub-trigger from message content
	userMessage := strings.TrimSpace(strings.TrimPrefix(message.Content, webhook.SubTrigger))

	s.logger.Info("Processing webhook message",
		"sub_trigger", webhook.SubTrigger,
		"webhook_url", webhook.URL,
		"message", userMessage)

	// Save incoming message
	if err := s.repository.Save(ctx, message); err != nil {
		s.logger.Error("Failed to save message", "error", err)
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Parse webhook timeout (default to 30s if not specified or invalid)
	timeout := 30 * time.Second
	if webhook.Timeout != "" {
		if parsedTimeout, err := time.ParseDuration(webhook.Timeout); err == nil {
			timeout = parsedTimeout
			s.logger.Debug("Using webhook timeout", "timeout", timeout)
		} else {
			s.logger.Warn("Invalid webhook timeout, using default 30s", "timeout_config", webhook.Timeout, "error", err)
		}
	}

	// Create context with timeout
	webhookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Call webhook with timeout context
	response, err := s.webhookClient.Call(webhookCtx, webhook.URL, userMessage)
	if err != nil {
		s.logger.Error("Failed to call webhook", "error", err, "url", webhook.URL)

		// Send user-friendly error message as a reply
		errorMsg := "Sorry, I cannot process this request right now due to a technical error. Please try again later."
		if err := s.whatsapp.SendReply(ctx, message.GroupJID, errorMsg, message.ID, message.Sender); err != nil {
			s.logger.Error("Failed to send error message", "error", err)
		}
		return fmt.Errorf("failed to call webhook: %w", err)
	}

	s.logger.Info("Webhook response received", "type", response.ContentType)

	// Send webhook response back to WhatsApp based on content type
	var responseContent string
	if response.ContentType == "image/jpeg" || response.ContentType == "image/png" {
		// Send as image
		s.logger.Info("Sending image response", "size", len(response.Content), "mime", response.ContentType)
		if err := s.whatsapp.SendImage(ctx, message.GroupJID, response.Content, response.ContentType, "", message.ID, message.Sender); err != nil {
			s.logger.Error("Failed to send image response", "error", err)
			return fmt.Errorf("failed to send image: %w", err)
		}
		responseContent = "[Image sent]"
	} else {
		// Format text response for WhatsApp
		formattedText := FormatWebhookResponse(response.TextContent)

		// Send as text reply
		if err := s.whatsapp.SendReply(ctx, message.GroupJID, formattedText, message.ID, message.Sender); err != nil {
			s.logger.Error("Failed to send webhook response", "error", err)
			return fmt.Errorf("failed to send message: %w", err)
		}
		responseContent = formattedText
	}

	// Save bot response
	botMessage := &domain.Message{
		ID:        fmt.Sprintf("bot-%d", message.Timestamp.Unix()),
		GroupJID:  message.GroupJID,
		Sender:    "bot",
		Content:   responseContent,
		Timestamp: message.Timestamp,
		IsFromBot: true,
	}

	if err := s.repository.Save(ctx, botMessage); err != nil {
		s.logger.Error("Failed to save bot message", "error", err)
	}

	return nil
}

// Start initializes the chat service
func (s *ChatService) Start(ctx context.Context) error {
	// Register message handler
	s.whatsapp.OnMessage(func(msg *domain.Message) {
		if err := s.ProcessMessage(ctx, msg); err != nil {
			s.logger.Error("Failed to process message", "error", err)
		}
	})

	s.logger.Info("Chat service started")
	return nil
}

// UpdateWebhooks updates the webhook configurations dynamically
func (s *ChatService) UpdateWebhooks(webhooks []domain.WebhookConfig) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.webhookConfigs = webhooks
	s.logger.Info("Webhooks updated", "count", len(webhooks))
}

// UpdateTriggerWords updates the trigger words dynamically
func (s *ChatService) UpdateTriggerWords(triggerWords []string) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.triggerWords = triggerWords
	s.logger.Info("Trigger words updated", "count", len(triggerWords))
}
