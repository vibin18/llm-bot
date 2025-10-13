package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
	"gopkg.in/yaml.v3"
)

// FileConfigStore implements ConfigStore using YAML files
type FileConfigStore struct {
	filePath  string
	config    *domain.Config
	mu        sync.RWMutex
	logger    *slog.Logger
	callbacks []func(*domain.Config)
	watcher   *fsnotify.Watcher
}

// NewFileConfigStore creates a new file-based config store
func NewFileConfigStore(filePath string, logger *slog.Logger) *FileConfigStore {
	return &FileConfigStore{
		filePath:  filePath,
		logger:    logger,
		callbacks: make([]func(*domain.Config), 0),
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

// Watch monitors config file for changes and reloads automatically
func (s *FileConfigStore) Watch(callback func(*domain.Config)) error {
	s.mu.Lock()
	s.callbacks = append(s.callbacks, callback)

	// Start watcher if not already started
	if s.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to create file watcher: %w", err)
		}
		s.watcher = watcher
		s.mu.Unlock()

		// Start watching in a goroutine
		go s.watchLoop()

		// Add the config file to watch
		if err := s.watcher.Add(s.filePath); err != nil {
			return fmt.Errorf("failed to watch config file: %w", err)
		}

		s.logger.Info("Config file watcher started", "file", s.filePath)
	} else {
		s.mu.Unlock()
	}

	return nil
}

// watchLoop monitors file changes and triggers reload
func (s *FileConfigStore) watchLoop() {
	debounceTimer := time.NewTimer(0)
	<-debounceTimer.C // Drain the initial timer

	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Only reload on write or create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Debounce: wait 500ms before reloading to avoid multiple rapid reloads
				debounceTimer.Reset(500 * time.Millisecond)
				go func() {
					<-debounceTimer.C
					s.reloadConfig()
				}()
			}

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			s.logger.Error("Config watcher error", "error", err)
		}
	}
}

// reloadConfig reloads the configuration and notifies callbacks
func (s *FileConfigStore) reloadConfig() {
	s.logger.Info("Reloading configuration from file", "file", s.filePath)

	config, err := s.Load()
	if err != nil {
		s.logger.Error("Failed to reload config", "error", err)
		return
	}

	s.logger.Info("Configuration reloaded successfully",
		"webhooks_count", len(config.Webhooks),
		"allowed_groups_count", len(config.WhatsApp.AllowedGroups))

	// Notify all callbacks
	s.mu.RLock()
	callbacks := make([]func(*domain.Config), len(s.callbacks))
	copy(callbacks, s.callbacks)
	s.mu.RUnlock()

	for _, callback := range callbacks {
		callback(config)
	}
}

// Close stops the file watcher
func (s *FileConfigStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.watcher != nil {
		return s.watcher.Close()
	}
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
