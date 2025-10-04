package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// MockLLMProvider is a mock implementation of LLMProvider
type MockLLMProvider struct {
	response string
	err      error
}

func (m *MockLLMProvider) Generate(ctx context.Context, request *domain.LLMRequest) (*domain.LLMResponse, error) {
	if m.err != nil {
		return &domain.LLMResponse{Error: m.err}, m.err
	}
	return &domain.LLMResponse{Content: m.response}, nil
}

func (m *MockLLMProvider) IsAvailable(ctx context.Context) bool {
	return m.err == nil
}

// MockMessageRepository is a mock implementation of MessageRepository
type MockMessageRepository struct {
	messages []*domain.Message
}

func (m *MockMessageRepository) Save(ctx context.Context, message *domain.Message) error {
	m.messages = append(m.messages, message)
	return nil
}

func (m *MockMessageRepository) GetByGroupJID(ctx context.Context, groupJID string, limit int) ([]*domain.Message, error) {
	var result []*domain.Message
	for _, msg := range m.messages {
		if msg.GroupJID == groupJID {
			result = append(result, msg)
		}
	}
	return result, nil
}

func (m *MockMessageRepository) GetAll(ctx context.Context) ([]*domain.Message, error) {
	return m.messages, nil
}

// MockWhatsAppClient is a mock implementation of WhatsAppClient
type MockWhatsAppClient struct {
	sentMessages []string
}

func (m *MockWhatsAppClient) Start(ctx context.Context) error { return nil }
func (m *MockWhatsAppClient) Stop(ctx context.Context) error  { return nil }

func (m *MockWhatsAppClient) SendMessage(ctx context.Context, groupJID, message string) error {
	m.sentMessages = append(m.sentMessages, message)
	return nil
}

func (m *MockWhatsAppClient) GetGroups(ctx context.Context) ([]*domain.Group, error) {
	return nil, nil
}

func (m *MockWhatsAppClient) GetAuthStatus(ctx context.Context) (*domain.AuthStatus, error) {
	return &domain.AuthStatus{IsAuthenticated: true}, nil
}

func (m *MockWhatsAppClient) OnMessage(handler func(*domain.Message)) {}

// MockGroupManager is a mock implementation of GroupManager
type MockGroupManager struct {
	allowedGroups map[string]bool
}

func (m *MockGroupManager) IsAllowed(groupJID string) bool {
	return m.allowedGroups[groupJID]
}

func (m *MockGroupManager) AddAllowedGroup(groupJID string) error {
	m.allowedGroups[groupJID] = true
	return nil
}

func (m *MockGroupManager) RemoveAllowedGroup(groupJID string) error {
	delete(m.allowedGroups, groupJID)
	return nil
}

func (m *MockGroupManager) GetAllowedGroups() []string {
	var groups []string
	for group := range m.allowedGroups {
		groups = append(groups, group)
	}
	return groups
}

func (m *MockGroupManager) UpdateAllowedGroups(groups []string) error {
	m.allowedGroups = make(map[string]bool)
	for _, group := range groups {
		m.allowedGroups[group] = true
	}
	return nil
}

func (m *MockGroupManager) SyncWithConfig() error {
	return nil
}

func TestChatService_ProcessMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name          string
		allowedGroups map[string]bool
		message       *domain.Message
		llmResponse   string
		wantProcessed bool
	}{
		{
			name:          "Process message from allowed group",
			allowedGroups: map[string]bool{"test-group@g.us": true},
			message: &domain.Message{
				ID:        "msg1",
				GroupJID:  "test-group@g.us",
				Sender:    "user@s.whatsapp.net",
				Content:   "Hello bot",
				Timestamp: time.Now(),
				IsFromBot: false,
			},
			llmResponse:   "Hello! How can I help you?",
			wantProcessed: true,
		},
		{
			name:          "Ignore message from non-allowed group",
			allowedGroups: map[string]bool{"allowed-group@g.us": true},
			message: &domain.Message{
				ID:        "msg2",
				GroupJID:  "other-group@g.us",
				Sender:    "user@s.whatsapp.net",
				Content:   "Hello bot",
				Timestamp: time.Now(),
				IsFromBot: false,
			},
			llmResponse:   "This should not be sent",
			wantProcessed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llmProvider := &MockLLMProvider{response: tt.llmResponse}
			repository := &MockMessageRepository{}
			whatsapp := &MockWhatsAppClient{}
			groupMgr := &MockGroupManager{allowedGroups: tt.allowedGroups}

			service := NewChatService(llmProvider, repository, whatsapp, groupMgr, []string{}, logger)

			ctx := context.Background()
			err := service.ProcessMessage(ctx, tt.message)

			if err != nil {
				t.Fatalf("ProcessMessage() error = %v", err)
			}

			if tt.wantProcessed {
				if len(whatsapp.sentMessages) != 1 {
					t.Errorf("Expected 1 sent message, got %d", len(whatsapp.sentMessages))
				}
				if len(whatsapp.sentMessages) > 0 && whatsapp.sentMessages[0] != tt.llmResponse {
					t.Errorf("Expected message %q, got %q", tt.llmResponse, whatsapp.sentMessages[0])
				}
			} else {
				if len(whatsapp.sentMessages) != 0 {
					t.Errorf("Expected 0 sent messages, got %d", len(whatsapp.sentMessages))
				}
			}
		})
	}
}
