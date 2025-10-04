package services

import (
	"fmt"
	"sync"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// GroupService manages WhatsApp groups
type GroupService struct {
	configStore   domain.ConfigStore
	allowedGroups map[string]bool
	mu            sync.RWMutex
}

// NewGroupService creates a new group service
func NewGroupService(configStore domain.ConfigStore) *GroupService {
	return &GroupService{
		configStore:   configStore,
		allowedGroups: make(map[string]bool),
	}
}

// IsAllowed checks if a group is allowed
func (s *GroupService) IsAllowed(groupJID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.allowedGroups[groupJID]
}

// AddAllowedGroup adds a group to the allowed list
func (s *GroupService) AddAllowedGroup(groupJID string) error {
	s.mu.Lock()
	s.allowedGroups[groupJID] = true
	groups := s.getAllowedGroupsLocked()
	s.mu.Unlock()

	return s.configStore.UpdateAllowedGroups(groups)
}

// RemoveAllowedGroup removes a group from the allowed list
func (s *GroupService) RemoveAllowedGroup(groupJID string) error {
	s.mu.Lock()
	delete(s.allowedGroups, groupJID)
	groups := s.getAllowedGroupsLocked()
	s.mu.Unlock()

	return s.configStore.UpdateAllowedGroups(groups)
}

// GetAllowedGroups returns all allowed groups
func (s *GroupService) GetAllowedGroups() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getAllowedGroupsLocked()
}

// SyncWithConfig synchronizes with configuration
func (s *GroupService) SyncWithConfig() error {
	groups, err := s.configStore.GetAllowedGroups()
	if err != nil {
		return fmt.Errorf("failed to get allowed groups from config: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.allowedGroups = make(map[string]bool)
	for _, group := range groups {
		s.allowedGroups[group] = true
	}

	return nil
}

// UpdateAllowedGroups updates the entire allowed groups list
func (s *GroupService) UpdateAllowedGroups(groups []string) error {
	s.mu.Lock()
	s.allowedGroups = make(map[string]bool)
	for _, group := range groups {
		s.allowedGroups[group] = true
	}
	s.mu.Unlock()

	return s.configStore.UpdateAllowedGroups(groups)
}

// getAllowedGroupsLocked returns allowed groups (must be called with lock held)
func (s *GroupService) getAllowedGroupsLocked() []string {
	groups := make([]string, 0, len(s.allowedGroups))
	for group := range s.allowedGroups {
		groups = append(groups, group)
	}
	return groups
}
