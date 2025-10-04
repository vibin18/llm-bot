package services

import (
	"testing"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// MockConfigStore is a mock implementation of ConfigStore
type MockConfigStore struct {
	config *domain.Config
}

func (m *MockConfigStore) Load() (*domain.Config, error) {
	return m.config, nil
}

func (m *MockConfigStore) Save(config *domain.Config) error {
	m.config = config
	return nil
}

func (m *MockConfigStore) UpdateAllowedGroups(groups []string) error {
	if m.config == nil {
		m.config = &domain.Config{}
	}
	m.config.WhatsApp.AllowedGroups = groups
	return nil
}

func (m *MockConfigStore) GetAllowedGroups() ([]string, error) {
	if m.config == nil {
		return []string{}, nil
	}
	return m.config.WhatsApp.AllowedGroups, nil
}

func (m *MockConfigStore) Watch(callback func(*domain.Config)) error {
	return nil
}

func TestGroupService_IsAllowed(t *testing.T) {
	configStore := &MockConfigStore{
		config: &domain.Config{
			WhatsApp: domain.WhatsAppConfig{
				AllowedGroups: []string{"group1@g.us", "group2@g.us"},
			},
		},
	}

	service := NewGroupService(configStore)
	service.SyncWithConfig()

	tests := []struct {
		name     string
		groupJID string
		want     bool
	}{
		{
			name:     "Allowed group returns true",
			groupJID: "group1@g.us",
			want:     true,
		},
		{
			name:     "Another allowed group returns true",
			groupJID: "group2@g.us",
			want:     true,
		},
		{
			name:     "Non-allowed group returns false",
			groupJID: "group3@g.us",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.IsAllowed(tt.groupJID)
			if got != tt.want {
				t.Errorf("IsAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupService_AddAllowedGroup(t *testing.T) {
	configStore := &MockConfigStore{
		config: &domain.Config{
			WhatsApp: domain.WhatsAppConfig{
				AllowedGroups: []string{"group1@g.us"},
			},
		},
	}

	service := NewGroupService(configStore)
	service.SyncWithConfig()

	err := service.AddAllowedGroup("group2@g.us")
	if err != nil {
		t.Fatalf("AddAllowedGroup() error = %v", err)
	}

	if !service.IsAllowed("group2@g.us") {
		t.Error("Expected group2@g.us to be allowed after adding")
	}

	groups, _ := configStore.GetAllowedGroups()
	if len(groups) != 2 {
		t.Errorf("Expected 2 allowed groups in config, got %d", len(groups))
	}
}

func TestGroupService_RemoveAllowedGroup(t *testing.T) {
	configStore := &MockConfigStore{
		config: &domain.Config{
			WhatsApp: domain.WhatsAppConfig{
				AllowedGroups: []string{"group1@g.us", "group2@g.us"},
			},
		},
	}

	service := NewGroupService(configStore)
	service.SyncWithConfig()

	err := service.RemoveAllowedGroup("group1@g.us")
	if err != nil {
		t.Fatalf("RemoveAllowedGroup() error = %v", err)
	}

	if service.IsAllowed("group1@g.us") {
		t.Error("Expected group1@g.us to not be allowed after removing")
	}

	if !service.IsAllowed("group2@g.us") {
		t.Error("Expected group2@g.us to still be allowed")
	}

	groups, _ := configStore.GetAllowedGroups()
	if len(groups) != 1 {
		t.Errorf("Expected 1 allowed group in config, got %d", len(groups))
	}
}

func TestGroupService_UpdateAllowedGroups(t *testing.T) {
	configStore := &MockConfigStore{
		config: &domain.Config{
			WhatsApp: domain.WhatsAppConfig{
				AllowedGroups: []string{},
			},
		},
	}

	service := NewGroupService(configStore)

	newGroups := []string{"groupA@g.us", "groupB@g.us", "groupC@g.us"}
	err := service.UpdateAllowedGroups(newGroups)
	if err != nil {
		t.Fatalf("UpdateAllowedGroups() error = %v", err)
	}

	for _, group := range newGroups {
		if !service.IsAllowed(group) {
			t.Errorf("Expected %s to be allowed", group)
		}
	}

	groups := service.GetAllowedGroups()
	if len(groups) != len(newGroups) {
		t.Errorf("Expected %d allowed groups, got %d", len(newGroups), len(groups))
	}
}
