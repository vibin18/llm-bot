package domain

import "time"

// Message represents a chat message
type Message struct {
	ID           string
	GroupJID     string
	Sender       string
	Content      string
	Timestamp    time.Time
	IsFromBot    bool
	IsReplyToBot bool // true if this is a reply to bot's message
}

// Group represents a WhatsApp group
type Group struct {
	JID          string `json:"jid"`
	Name         string `json:"name"`
	IsAllowed    bool   `json:"is_allowed"`
	Participants int    `json:"participants"`
}

// Config represents application configuration
type Config struct {
	App      AppConfig      `yaml:"app"`
	WhatsApp WhatsAppConfig `yaml:"whatsapp"`
	Ollama   OllamaConfig   `yaml:"ollama"`
	Storage  StorageConfig  `yaml:"storage"`
	Webhooks []WebhookConfig `yaml:"webhooks"`
}

// AppConfig contains application-level settings
type AppConfig struct {
	Name     string `yaml:"name"`
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"log_level"`
}

// WhatsAppConfig contains WhatsApp-specific settings
type WhatsAppConfig struct {
	SessionPath   string   `yaml:"session_path"`
	AllowedGroups []string `yaml:"allowed_groups"`
	TriggerWords  []string `yaml:"trigger_words"`
}

// OllamaConfig contains Ollama LLM settings
type OllamaConfig struct {
	URL         string  `yaml:"url"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	Timeout     string  `yaml:"timeout"`
}

// StorageConfig contains storage settings
type StorageConfig struct {
	Type string `yaml:"type"`
}

// WebhookConfig contains webhook settings
type WebhookConfig struct {
	SubTrigger string `yaml:"sub_trigger" json:"sub_trigger"`
	URL        string `yaml:"url" json:"url"`
	Timeout    string `yaml:"timeout" json:"timeout"` // e.g., "60s", "2m"
}

// LLMRequest represents a request to the LLM
type LLMRequest struct {
	Prompt  string
	Context []Message
}

// LLMResponse represents a response from the LLM
type LLMResponse struct {
	Content string
	Error   error
}

// AuthStatus represents WhatsApp authentication status
type AuthStatus struct {
	IsAuthenticated bool   `json:"is_authenticated"`
	QRCode          string `json:"qr_code"`
	Error           error  `json:"error"`
}
