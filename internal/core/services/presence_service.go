package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// PresenceService tracks WhatsApp contact presence and exposes Prometheus metrics
type PresenceService struct {
	contacts          map[string]*domain.ContactPresence
	mu                sync.RWMutex
	logger            *slog.Logger
	subscriptionMgr   *SubscriptionManager

	// Prometheus metrics
	onlineGauge       *prometheus.GaugeVec
	statusChanges     *prometheus.CounterVec
	lastSeenGauge     *prometheus.GaugeVec
}

// NewPresenceService creates a new presence tracking service
func NewPresenceService(logger *slog.Logger) *PresenceService {
	// Create Prometheus metrics
	onlineGauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "whatsapp_contact_online",
			Help: "Current online status of WhatsApp contacts (1 = online, 0 = offline)",
		},
		[]string{"jid", "name"},
	)

	statusChanges := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "whatsapp_contact_status_changes_total",
			Help: "Total number of status changes for WhatsApp contacts",
		},
		[]string{"jid", "name", "status"},
	)

	lastSeenGauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "whatsapp_contact_last_seen_timestamp_seconds",
			Help: "Unix timestamp of when the contact was last seen online",
		},
		[]string{"jid", "name"},
	)

	subscriptionMgr := NewSubscriptionManager(logger)

	return &PresenceService{
		contacts:        make(map[string]*domain.ContactPresence),
		logger:          logger,
		subscriptionMgr: subscriptionMgr,
		onlineGauge:     onlineGauge,
		statusChanges:   statusChanges,
		lastSeenGauge:   lastSeenGauge,
	}
}

// UpdatePresence updates the presence status for a contact
func (s *PresenceService) UpdatePresence(event *domain.PresenceEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Record event in subscription manager (for health tracking)
	if s.subscriptionMgr != nil {
		s.subscriptionMgr.RecordEvent(event.JID)
	}

	now := time.Now()
	contact, exists := s.contacts[event.JID]

	if !exists {
		// New contact
		contact = &domain.ContactPresence{
			JID:              event.JID,
			IsOnline:         event.IsOnline,
			LastSeen:         event.Timestamp,
			LastStatusChange: now,
		}
		s.contacts[event.JID] = contact
		s.logger.Info("New contact tracked", "jid", event.JID, "online", event.IsOnline)
	} else {
		// Existing contact - only update if status changed
		if contact.IsOnline != event.IsOnline {
			contact.IsOnline = event.IsOnline
			contact.LastStatusChange = now

			// Update last seen when going offline
			if !event.IsOnline {
				contact.LastSeen = event.Timestamp
			}

			s.logger.Info("Contact status changed",
				"jid", event.JID,
				"online", event.IsOnline,
				"was_online", !event.IsOnline)
		}
	}

	// Update Prometheus metrics
	s.updateMetrics(contact)
}

// updateMetrics updates Prometheus metrics for a contact
func (s *PresenceService) updateMetrics(contact *domain.ContactPresence) {
	labels := prometheus.Labels{
		"jid":  contact.JID,
		"name": contact.Name,
	}

	// Update online/offline gauge
	if contact.IsOnline {
		s.onlineGauge.With(labels).Set(1)
		s.statusChanges.With(prometheus.Labels{
			"jid":    contact.JID,
			"name":   contact.Name,
			"status": "online",
		}).Inc()
	} else {
		s.onlineGauge.With(labels).Set(0)
		s.statusChanges.With(prometheus.Labels{
			"jid":    contact.JID,
			"name":   contact.Name,
			"status": "offline",
		}).Inc()

		// Update last seen timestamp
		s.lastSeenGauge.With(labels).Set(float64(contact.LastSeen.Unix()))
	}
}

// InitializeContact initializes a contact for tracking (called when subscribing)
func (s *PresenceService) InitializeContact(jid string, name ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only initialize if not already tracked
	if _, exists := s.contacts[jid]; !exists {
		now := time.Now()
		contactName := jid // Default to JID
		if len(name) > 0 && name[0] != "" {
			contactName = name[0]
		}

		contact := &domain.ContactPresence{
			JID:              jid,
			Name:             contactName,
			IsOnline:         false,
			LastSeen:         now,
			LastStatusChange: now,
		}
		s.contacts[jid] = contact

		// Initialize metrics
		s.updateMetrics(contact)

		s.logger.Debug("Initialized contact for tracking", "jid", jid, "name", contactName)
	}
}

// GetPresence retrieves the presence status for a specific contact
func (s *PresenceService) GetPresence(jid string) (*domain.ContactPresence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contact, exists := s.contacts[jid]
	return contact, exists
}

// GetAllPresences retrieves all tracked contact presences
func (s *PresenceService) GetAllPresences() []*domain.ContactPresence {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*domain.ContactPresence, 0, len(s.contacts))
	for _, contact := range s.contacts {
		// Create a copy to avoid concurrent access issues
		contactCopy := *contact
		result = append(result, &contactCopy)
	}

	return result
}

// GetOnlineCount returns the number of currently online contacts
func (s *PresenceService) GetOnlineCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, contact := range s.contacts {
		if contact.IsOnline {
			count++
		}
	}

	return count
}

// SetContactName sets the name for a contact (optional, for better metrics labels)
func (s *PresenceService) SetContactName(jid, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if contact, exists := s.contacts[jid]; exists {
		contact.Name = name
		s.updateMetrics(contact)
	}
}

// RemoveContact removes a contact from tracking
func (s *PresenceService) RemoveContact(jid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	contact, exists := s.contacts[jid]
	if !exists {
		return false
	}

	// Remove from tracking
	delete(s.contacts, jid)

	// Remove from metrics
	s.onlineGauge.DeleteLabelValues(contact.JID, contact.Name)
	s.lastSeenGauge.DeleteLabelValues(contact.JID, contact.Name)

	s.logger.Info("Removed contact from tracking", "jid", jid)
	return true
}

// Start starts the presence service (placeholder for future cleanup tasks)
func (s *PresenceService) Start(ctx context.Context) error {
	s.logger.Info("Presence tracking service started")

	// Optional: Add periodic cleanup of stale contacts
	go s.cleanupRoutine(ctx)

	return nil
}

// GetSubscriptionManager returns the subscription manager for external use
func (s *PresenceService) GetSubscriptionManager() *SubscriptionManager {
	return s.subscriptionMgr
}

// cleanupRoutine periodically cleans up very old presence data
func (s *PresenceService) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupStaleContacts()
		case <-ctx.Done():
			return
		}
	}
}

// cleanupStaleContacts removes contacts that haven't been seen in 30 days
func (s *PresenceService) cleanupStaleContacts() {
	s.mu.Lock()
	defer s.mu.Unlock()

	threshold := time.Now().Add(-30 * 24 * time.Hour)
	removed := 0

	for jid, contact := range s.contacts {
		if contact.LastSeen.Before(threshold) && contact.LastStatusChange.Before(threshold) {
			delete(s.contacts, jid)
			removed++

			// Remove from metrics
			s.onlineGauge.DeleteLabelValues(contact.JID, contact.Name)
			s.lastSeenGauge.DeleteLabelValues(contact.JID, contact.Name)
		}
	}

	if removed > 0 {
		s.logger.Info("Cleaned up stale contacts", "count", removed)
	}
}
