package services

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// SubscriptionManager manages presence subscriptions with rate limiting protection
type SubscriptionManager struct {
	subscriptions     map[string]*SubscriptionInfo
	mu                sync.RWMutex
	logger            *slog.Logger

	// Rate limiting
	subscriptionQueue chan string
	batchSize         int
	batchDelay        time.Duration
	resubscribeAfter  time.Duration
}

// SubscriptionInfo tracks subscription metadata
type SubscriptionInfo struct {
	JID              string
	SubscribedAt     time.Time
	LastEventAt      time.Time
	Priority         int  // 1=high, 2=medium, 3=low
	FailCount        int
	NextRetry        time.Time
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(logger *slog.Logger) *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions:    make(map[string]*SubscriptionInfo),
		logger:           logger,
		subscriptionQueue: make(chan string, 1000),
		batchSize:        20,  // Subscribe to 20 contacts per batch
		batchDelay:       5 * time.Second,  // 5 second delay between batches
		resubscribeAfter: 24 * time.Hour,   // Re-subscribe after 24 hours only if no events
	}
}

// Start starts the subscription manager
func (m *SubscriptionManager) Start(ctx context.Context, subscribeFn func(string) error) error {
	m.logger.Info("Starting subscription manager",
		"batch_size", m.batchSize,
		"batch_delay", m.batchDelay)

	// Process subscription queue with batching
	go m.processBatchedSubscriptions(ctx, subscribeFn)

	// Periodic health check and re-subscription (less aggressive)
	go m.healthCheckRoutine(ctx, subscribeFn)

	return nil
}

// QueueSubscription queues a contact for subscription
func (m *SubscriptionManager) QueueSubscription(jid string, priority int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already subscribed recently
	if info, exists := m.subscriptions[jid]; exists {
		// If subscribed within last hour, skip
		if time.Since(info.SubscribedAt) < 1*time.Hour {
			m.logger.Debug("Skipping recent subscription", "jid", jid, "age", time.Since(info.SubscribedAt))
			return
		}
	}

	// Create or update subscription info
	m.subscriptions[jid] = &SubscriptionInfo{
		JID:          jid,
		SubscribedAt: time.Time{}, // Will be set when actually subscribed
		LastEventAt:  time.Now(),
		Priority:     priority,
		FailCount:    0,
	}

	// Add to queue
	select {
	case m.subscriptionQueue <- jid:
		m.logger.Debug("Queued subscription", "jid", jid, "priority", priority)
	default:
		m.logger.Warn("Subscription queue full, dropping", "jid", jid)
	}
}

// processBatchedSubscriptions processes subscriptions in batches to avoid rate limiting
func (m *SubscriptionManager) processBatchedSubscriptions(ctx context.Context, subscribeFn func(string) error) {
	ticker := time.NewTicker(m.batchDelay)
	defer ticker.Stop()

	batch := make([]string, 0, m.batchSize)

	for {
		select {
		case <-ctx.Done():
			return

		case jid := <-m.subscriptionQueue:
			batch = append(batch, jid)

			// Process batch when full or after delay
			if len(batch) >= m.batchSize {
				m.subscribeBatch(batch, subscribeFn)
				batch = make([]string, 0, m.batchSize)
				ticker.Reset(m.batchDelay)
			}

		case <-ticker.C:
			// Process partial batch
			if len(batch) > 0 {
				m.subscribeBatch(batch, subscribeFn)
				batch = make([]string, 0, m.batchSize)
			}
		}
	}
}

// subscribeBatch subscribes to a batch of contacts
func (m *SubscriptionManager) subscribeBatch(jids []string, subscribeFn func(string) error) {
	m.logger.Info("Processing subscription batch", "count", len(jids))

	for _, jid := range jids {
		// Check if we should retry
		m.mu.RLock()
		info := m.subscriptions[jid]
		m.mu.RUnlock()

		if info != nil && time.Now().Before(info.NextRetry) {
			m.logger.Debug("Skipping subscription (backoff)", "jid", jid, "retry_at", info.NextRetry)
			continue
		}

		// Attempt subscription
		err := subscribeFn(jid)

		m.mu.Lock()
		if err != nil {
			// Exponential backoff on failure
			info.FailCount++
			backoff := time.Duration(1<<uint(info.FailCount)) * time.Minute // 2^n minutes
			if backoff > 1*time.Hour {
				backoff = 1 * time.Hour
			}
			info.NextRetry = time.Now().Add(backoff)

			m.logger.Warn("Subscription failed, backing off",
				"jid", jid,
				"fail_count", info.FailCount,
				"retry_after", backoff)
		} else {
			// Success
			info.SubscribedAt = time.Now()
			info.FailCount = 0
			info.NextRetry = time.Time{}

			m.logger.Debug("Subscription successful", "jid", jid)
		}
		m.mu.Unlock()

		// Small delay between individual subscriptions in batch
		time.Sleep(200 * time.Millisecond)
	}
}

// RecordEvent records that we received an event for a contact
func (m *SubscriptionManager) RecordEvent(jid string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, exists := m.subscriptions[jid]; exists {
		info.LastEventAt = time.Now()
		m.logger.Debug("Recorded event", "jid", jid)
	}
}

// healthCheckRoutine checks for stale subscriptions and re-subscribes if needed
func (m *SubscriptionManager) healthCheckRoutine(ctx context.Context, subscribeFn func(string) error) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.performHealthCheck(subscribeFn)
		}
	}
}

// performHealthCheck checks for stale subscriptions
func (m *SubscriptionManager) performHealthCheck(subscribeFn func(string) error) {
	m.mu.RLock()
	staleJIDs := make([]string, 0)

	for jid, info := range m.subscriptions {
		// Only re-subscribe if:
		// 1. No events received in resubscribeAfter duration
		// 2. Originally subscribed more than resubscribeAfter ago
		if time.Since(info.LastEventAt) > m.resubscribeAfter &&
		   time.Since(info.SubscribedAt) > m.resubscribeAfter {
			staleJIDs = append(staleJIDs, jid)
		}
	}
	m.mu.RUnlock()

	if len(staleJIDs) > 0 {
		m.logger.Info("Found stale subscriptions", "count", len(staleJIDs))

		// Re-queue stale subscriptions (they'll go through batching)
		for _, jid := range staleJIDs {
			m.QueueSubscription(jid, 2) // Medium priority
		}
	}
}

// GetStats returns subscription statistics
func (m *SubscriptionManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	activeCount := 0
	staleCount := 0
	failedCount := 0

	for _, info := range m.subscriptions {
		if info.FailCount > 0 {
			failedCount++
		} else if time.Since(info.LastEventAt) > m.resubscribeAfter {
			staleCount++
		} else {
			activeCount++
		}
	}

	return map[string]interface{}{
		"total_subscriptions": len(m.subscriptions),
		"active":              activeCount,
		"stale":               staleCount,
		"failed":              failedCount,
		"queue_length":        len(m.subscriptionQueue),
	}
}

// SetBatchConfig allows customizing batch configuration
func (m *SubscriptionManager) SetBatchConfig(batchSize int, batchDelay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.batchSize = batchSize
	m.batchDelay = batchDelay

	m.logger.Info("Updated batch configuration",
		"batch_size", batchSize,
		"batch_delay", batchDelay)
}
