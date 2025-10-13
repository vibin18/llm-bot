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

// WebhookResponse represents a response from a webhook
type WebhookResponse struct {
	ContentType string // "text", "image/jpeg", "image/png"
	Content     []byte // Raw content (text or image data)
	TextContent string // Convenience field for text responses
}

// Schedule represents a scheduled webhook trigger
type Schedule struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	GroupJID     string     `json:"group_jid"`
	WebhookURL   string     `json:"webhook_url"`
	ScheduleType string     `json:"schedule_type"`            // "weekly", "yearly", "once"
	DayOfWeek    *int       `json:"day_of_week,omitempty"`    // 0 = Sunday, 6 = Saturday (for weekly)
	Month        *int       `json:"month,omitempty"`          // 1-12 (for yearly)
	DayOfMonth   *int       `json:"day_of_month,omitempty"`   // 1-31 (for yearly)
	Hour         int        `json:"hour"`                     // 0-23
	Minute       int        `json:"minute"`                   // 0-59
	SpecificDate *time.Time `json:"specific_date,omitempty"`  // Specific date for one-time schedules
	Enabled      bool       `json:"enabled"`
	LastRun      *time.Time `json:"last_run,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ScheduleExecution represents a log of schedule execution
type ScheduleExecution struct {
	ID         string    `json:"id"`
	ScheduleID string    `json:"schedule_id"`
	ExecutedAt time.Time `json:"executed_at"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	Response   string    `json:"response,omitempty"`
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
