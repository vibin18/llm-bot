package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
	"gopkg.in/yaml.v3"
)

// FileConfigStore implements ConfigStore using YAML files
type FileConfigStore struct {
	filePath string
	config   *domain.Config
	mu       sync.RWMutex
}

// NewFileConfigStore creates a new file-based config store
func NewFileConfigStore(filePath string) *FileConfigStore {
	return &FileConfigStore{
		filePath: filePath,
	}
}

// Load reads configuration from file and applies environment overrides
func (s *FileConfigStore) Load() (*domain.Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read YAML file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config domain.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	s.applyEnvOverrides(&config)

	// Validate configuration
	if err := s.validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	s.config = &config
	return &config, nil
}

// Save writes configuration to file
func (s *FileConfigStore) Save(config *domain.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate before saving
	if err := s.validate(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	s.config = config
	return nil
}

// UpdateAllowedGroups updates the allowed groups list
func (s *FileConfigStore) UpdateAllowedGroups(groups []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config == nil {
		return fmt.Errorf("config not loaded")
	}

	s.config.WhatsApp.AllowedGroups = groups
	return s.Save(s.config)
}

// GetAllowedGroups returns the current allowed groups
func (s *FileConfigStore) GetAllowedGroups() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	return s.config.WhatsApp.AllowedGroups, nil
}

// Watch monitors config file for changes (simplified implementation)
func (s *FileConfigStore) Watch(callback func(*domain.Config)) error {
	// Note: Full implementation would use fsnotify or similar
	// For now, this is a placeholder for hot-reload capability
	return nil
}

// applyEnvOverrides applies environment variable overrides
func (s *FileConfigStore) applyEnvOverrides(config *domain.Config) {
	if val := os.Getenv("APP_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.App.Port = port
		}
	}

	if val := os.Getenv("APP_LOG_LEVEL"); val != "" {
		config.App.LogLevel = val
	}

	if val := os.Getenv("WHATSAPP_SESSION_PATH"); val != "" {
		config.WhatsApp.SessionPath = val
	}

	if val := os.Getenv("OLLAMA_URL"); val != "" {
		config.Ollama.URL = val
	}

	if val := os.Getenv("OLLAMA_MODEL"); val != "" {
		config.Ollama.Model = val
	}

	if val := os.Getenv("OLLAMA_TEMPERATURE"); val != "" {
		if temp, err := strconv.ParseFloat(val, 64); err == nil {
			config.Ollama.Temperature = temp
		}
	}
}

// validate validates configuration values
func (s *FileConfigStore) validate(config *domain.Config) error {
	if config.App.Port < 1 || config.App.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.App.Port)
	}

	if config.App.LogLevel == "" {
		return fmt.Errorf("log level cannot be empty")
	}

	if config.Ollama.URL == "" {
		return fmt.Errorf("ollama URL cannot be empty")
	}

	if config.Ollama.Model == "" {
		return fmt.Errorf("ollama model cannot be empty")
	}

	if config.Ollama.Temperature < 0 || config.Ollama.Temperature > 2 {
		return fmt.Errorf("invalid temperature: %f (must be between 0 and 2)", config.Ollama.Temperature)
	}

	return nil
}
