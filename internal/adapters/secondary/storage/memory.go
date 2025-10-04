package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// MemoryRepository implements MessageRepository using in-memory storage
type MemoryRepository struct {
	messages map[string][]*domain.Message // groupJID -> messages
	mu       sync.RWMutex
}

// NewMemoryRepository creates a new in-memory message repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		messages: make(map[string][]*domain.Message),
	}
}

// Save stores a message
func (r *MemoryRepository) Save(ctx context.Context, message *domain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if message.GroupJID == "" {
		return fmt.Errorf("group JID is required")
	}

	r.messages[message.GroupJID] = append(r.messages[message.GroupJID], message)
	return nil
}

// GetByGroupJID retrieves messages for a specific group
func (r *MemoryRepository) GetByGroupJID(ctx context.Context, groupJID string, limit int) ([]*domain.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	messages, exists := r.messages[groupJID]
	if !exists {
		return []*domain.Message{}, nil
	}

	// Return last N messages
	start := 0
	if len(messages) > limit && limit > 0 {
		start = len(messages) - limit
	}

	result := make([]*domain.Message, len(messages)-start)
	copy(result, messages[start:])

	return result, nil
}

// GetAll retrieves all messages
func (r *MemoryRepository) GetAll(ctx context.Context) ([]*domain.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Message
	for _, msgs := range r.messages {
		result = append(result, msgs...)
	}

	return result, nil
}
