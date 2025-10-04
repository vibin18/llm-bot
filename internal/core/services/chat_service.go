package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// ChatService handles chat message processing
type ChatService struct {
	llmProvider  domain.LLMProvider
	repository   domain.MessageRepository
	whatsapp     domain.WhatsAppClient
	groupMgr     domain.GroupManager
	triggerWords []string
	logger       *slog.Logger
}

// NewChatService creates a new chat service
func NewChatService(
	llmProvider domain.LLMProvider,
	repository domain.MessageRepository,
	whatsapp domain.WhatsAppClient,
	groupMgr domain.GroupManager,
	triggerWords []string,
	logger *slog.Logger,
) *ChatService {
	return &ChatService{
		llmProvider:  llmProvider,
		repository:   repository,
		whatsapp:     whatsapp,
		groupMgr:     groupMgr,
		triggerWords: triggerWords,
		logger:       logger,
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
	if len(s.triggerWords) > 0 && !message.IsReplyToBot {
		trimmedContent := strings.TrimSpace(message.Content)
		triggered := false
		var matchedTrigger string

		for _, trigger := range s.triggerWords {
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
				"triggers", s.triggerWords,
				"content", message.Content)
			return nil
		}

		s.logger.Debug("Message triggered", "trigger", matchedTrigger)
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
		return fmt.Errorf("failed to generate response: %w", err)
	}

	s.logger.Info("Generated response", "content", response.Content)

	// Send response back to WhatsApp
	if err := s.whatsapp.SendMessage(ctx, message.GroupJID, response.Content); err != nil {
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
