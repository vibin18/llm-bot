package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// Handlers contains HTTP request handlers
type Handlers struct {
	whatsapp  domain.WhatsAppClient
	groupMgr  domain.GroupManager
	logger    *slog.Logger
}

// NewHandlers creates new HTTP handlers
func NewHandlers(whatsapp domain.WhatsAppClient, groupMgr domain.GroupManager, logger *slog.Logger) *Handlers {
	return &Handlers{
		whatsapp: whatsapp,
		groupMgr: groupMgr,
		logger:   logger,
	}
}

// GetGroups returns all WhatsApp groups
func (h *Handlers) GetGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.whatsapp.GetGroups(r.Context())
	if err != nil {
		h.logger.Error("Failed to get groups", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		Groups []string `json:"groups"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.groupMgr.UpdateAllowedGroups(req.Groups); err != nil {
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
