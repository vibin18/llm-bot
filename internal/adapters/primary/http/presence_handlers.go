package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vibin/whatsapp-llm-bot/internal/core/services"
)

// PresenceHandlers handles presence tracking HTTP requests
type PresenceHandlers struct {
	presenceService *services.PresenceService
	subscribeFunc   func(string, int) error
}

// NewPresenceHandlers creates new presence handlers
func NewPresenceHandlers(presenceService *services.PresenceService, subscribeFunc func(string, int) error) *PresenceHandlers {
	return &PresenceHandlers{
		presenceService: presenceService,
		subscribeFunc:   subscribeFunc,
	}
}

// GetAllPresences returns all tracked presences
func (h *PresenceHandlers) GetAllPresences(w http.ResponseWriter, r *http.Request) {
	presences := h.presenceService.GetAllPresences()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(presences)
}

// GetPresence returns presence for a specific contact
func (h *PresenceHandlers) GetPresence(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jid := vars["jid"]

	presence, exists := h.presenceService.GetPresence(jid)
	if !exists {
		http.Error(w, "Contact not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(presence)
}

// GetPresenceStats returns presence statistics
func (h *PresenceHandlers) GetPresenceStats(w http.ResponseWriter, r *http.Request) {
	presences := h.presenceService.GetAllPresences()
	onlineCount := h.presenceService.GetOnlineCount()

	stats := map[string]interface{}{
		"total_contacts": len(presences),
		"online_count":   onlineCount,
		"offline_count":  len(presences) - onlineCount,
	}

	// Add subscription manager stats if available
	if subMgr := h.presenceService.GetSubscriptionManager(); subMgr != nil {
		subStats := subMgr.GetStats()
		for k, v := range subStats {
			stats["subscription_"+k] = v
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// SubscribeToContact subscribes to a contact's presence
func (h *PresenceHandlers) SubscribeToContact(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JID      string `json:"jid"`
		Priority int    `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.JID == "" {
		http.Error(w, "JID is required", http.StatusBadRequest)
		return
	}

	// Default priority
	if req.Priority == 0 {
		req.Priority = 2 // Medium
	}

	// Queue subscription through the manager
	if subMgr := h.presenceService.GetSubscriptionManager(); subMgr != nil {
		subMgr.QueueSubscription(req.JID, req.Priority)
	} else if h.subscribeFunc != nil {
		// Fallback to direct subscription
		err := h.subscribeFunc(req.JID, req.Priority)
		if err != nil {
			http.Error(w, "Failed to subscribe: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "queued",
		"message": "Contact queued for presence subscription",
	})
}

// UnsubscribeFromContact unsubscribes from a contact's presence
func (h *PresenceHandlers) UnsubscribeFromContact(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jid := vars["jid"]

	if jid == "" {
		http.Error(w, "JID is required", http.StatusBadRequest)
		return
	}

	// Remove contact from presence tracking
	removed := h.presenceService.RemoveContact(jid)

	if !removed {
		http.Error(w, "Contact not found in tracking", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Contact removed from tracking",
	})
}

// BulkSubscribe subscribes to multiple contacts
func (h *PresenceHandlers) BulkSubscribe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JIDs     []string `json:"jids"`
		Priority int      `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.JIDs) == 0 {
		http.Error(w, "At least one JID is required", http.StatusBadRequest)
		return
	}

	// Default priority
	if req.Priority == 0 {
		req.Priority = 2 // Medium
	}

	// Queue all subscriptions
	if subMgr := h.presenceService.GetSubscriptionManager(); subMgr != nil {
		for _, jid := range req.JIDs {
			subMgr.QueueSubscription(jid, req.Priority)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "queued",
		"message": "Contacts queued for presence subscription",
		"count":   len(req.JIDs),
	})
}
