package domain

import "context"

// MessageRepository defines the interface for message storage
type MessageRepository interface {
	Save(ctx context.Context, message *Message) error
	GetByGroupJID(ctx context.Context, groupJID string, limit int) ([]*Message, error)
	GetAll(ctx context.Context) ([]*Message, error)
}

// LLMProvider defines the interface for LLM interactions
type LLMProvider interface {
	Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error)
	IsAvailable(ctx context.Context) bool
}

// WhatsAppClient defines the interface for WhatsApp operations
type WhatsAppClient interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SendMessage(ctx context.Context, groupJID, message string) error
	SendReply(ctx context.Context, groupJID, message, replyToMessageID, quotedSender string) error
	SendImage(ctx context.Context, groupJID string, imageData []byte, mimeType, caption, replyToMessageID, quotedSender string) error
	GetGroups(ctx context.Context) ([]*Group, error)
	GetAuthStatus(ctx context.Context) (*AuthStatus, error)
	OnMessage(handler func(*Message))
}

// ConfigStore defines the interface for configuration management
type ConfigStore interface {
	Load() (*Config, error)
	Save(config *Config) error
	UpdateAllowedGroups(groups []string) error
	GetAllowedGroups() ([]string, error)
	Watch(callback func(*Config)) error
}

// GroupManager defines the interface for group management
type GroupManager interface {
	IsAllowed(groupJID string) bool
	AddAllowedGroup(groupJID string) error
	RemoveAllowedGroup(groupJID string) error
	GetAllowedGroups() []string
	UpdateAllowedGroups(groups []string) error
	SyncWithConfig() error
}

// WebhookClient defines the interface for webhook interactions
type WebhookClient interface {
	Call(ctx context.Context, url string, message string) (*WebhookResponse, error)
}
