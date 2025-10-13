package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// Handlers contains HTTP request handlers
type Handlers struct {
	whatsapp    domain.WhatsAppClient
	groupMgr    domain.GroupManager
	configStore domain.ConfigStore
	logger      *slog.Logger
}

// NewHandlers creates new HTTP handlers
func NewHandlers(whatsapp domain.WhatsAppClient, groupMgr domain.GroupManager, configStore domain.ConfigStore, logger *slog.Logger) *Handlers {
	return &Handlers{
		whatsapp:    whatsapp,
		groupMgr:    groupMgr,
		configStore: configStore,
		logger:      logger,
	}
}

// GetGroups returns all WhatsApp groups
func (h *Handlers) GetGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.whatsapp.GetGroups(r.Context())
	if err != nil {
		// If WhatsApp is not connected, return allowed groups from config
		h.logger.Debug("WhatsApp not connected, returning allowed groups from config", "error", err)

		allowedGroups := h.groupMgr.GetAllowedGroups()

		// Convert allowed groups to group list format
		groupList := make([]map[string]interface{}, 0, len(allowedGroups))
		for _, jid := range allowedGroups {
			groupList = append(groupList, map[string]interface{}{
				"jid":  jid,
				"name": jid, // Use JID as name when we can't fetch real names
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groupList)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

// GetAllowedGroups returns currently allowed groups
func (h *Handlers) GetAllowedGroups(w http.ResponseWriter, r *http.Request) {
	groups := h.groupMgr.GetAllowedGroups()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed_groups": groups,
	})
}

// UpdateAllowedGroups updates the allowed groups list
func (h *Handlers) UpdateAllowedGroups(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AllowedGroups []string `json:"allowed_groups"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.groupMgr.UpdateAllowedGroups(req.AllowedGroups); err != nil {
		h.logger.Error("Failed to update allowed groups", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Allowed groups updated successfully",
	})
}

// GetStatus returns bot status and connection state
func (h *Handlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	authStatus, err := h.whatsapp.GetAuthStatus(r.Context())
	if err != nil {
		h.logger.Error("Failed to get auth status", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authStatus)
}

// GetQRCode triggers QR code generation for authentication
func (h *Handlers) GetQRCode(w http.ResponseWriter, r *http.Request) {
	authStatus, err := h.whatsapp.GetAuthStatus(r.Context())
	if err != nil {
		h.logger.Error("Failed to get auth status", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if authStatus.IsAuthenticated {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"message":       "Already authenticated",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"qr_code": authStatus.QRCode,
	})
}

// HealthCheck returns health status
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
	})
}

// GetWebhooks returns all configured webhooks
func (h *Handlers) GetWebhooks(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.configStore.Load()
	if err != nil {
		h.logger.Error("Failed to load config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"webhooks": cfg.Webhooks,
	})
}

// AddWebhook adds a new webhook configuration
func (h *Handlers) AddWebhook(w http.ResponseWriter, r *http.Request) {
	var webhook domain.WebhookConfig
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate webhook
	if webhook.SubTrigger == "" || webhook.URL == "" {
		http.Error(w, "sub_trigger and url are required", http.StatusBadRequest)
		return
	}

	h.logger.Debug("Adding webhook", "sub_trigger", webhook.SubTrigger, "url", webhook.URL)

	cfg, err := h.configStore.Load()
	if err != nil {
		h.logger.Error("Failed to load config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if sub_trigger already exists
	for _, wh := range cfg.Webhooks {
		if wh.SubTrigger == webhook.SubTrigger {
			http.Error(w, "sub_trigger already exists", http.StatusConflict)
			return
		}
	}

	cfg.Webhooks = append(cfg.Webhooks, webhook)

	if err := h.configStore.Save(cfg); err != nil {
		h.logger.Error("Failed to save config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Webhook added successfully",
		"webhook": webhook,
	})
}

// DeleteWebhook removes a webhook configuration
func (h *Handlers) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	// Get sub_trigger from query parameters
	subTrigger := r.URL.Query().Get("sub_trigger")
	if subTrigger == "" {
		http.Error(w, "sub_trigger query parameter is required", http.StatusBadRequest)
		return
	}

	cfg, err := h.configStore.Load()
	if err != nil {
		h.logger.Error("Failed to load config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Find and remove webhook
	found := false
	newWebhooks := make([]domain.WebhookConfig, 0)
	for _, webhook := range cfg.Webhooks {
		if webhook.SubTrigger != subTrigger {
			newWebhooks = append(newWebhooks, webhook)
		} else {
			found = true
		}
	}

	if !found {
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	cfg.Webhooks = newWebhooks

	if err := h.configStore.Save(cfg); err != nil {
		h.logger.Error("Failed to save config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Webhook deleted", "sub_trigger", subTrigger)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Webhook deleted successfully",
	})
}
