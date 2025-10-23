package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
)

// SchedulerService manages scheduled webhook triggers
type SchedulerService struct {
	repository    domain.ScheduleRepository
	webhookClient domain.WebhookClient
	whatsapp      domain.WhatsAppClient
	logger        *slog.Logger
	ticker        *time.Ticker
	stopChan      chan struct{}
	running       bool
	mu            sync.RWMutex
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(
	repository domain.ScheduleRepository,
	webhookClient domain.WebhookClient,
	whatsapp domain.WhatsAppClient,
	logger *slog.Logger,
) *SchedulerService {
	return &SchedulerService{
		repository:    repository,
		webhookClient: webhookClient,
		whatsapp:      whatsapp,
		logger:        logger,
		stopChan:      make(chan struct{}),
	}
}

// Start starts the scheduler
func (s *SchedulerService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.logger.Info("Starting scheduler service")

	// Check schedules every minute
	s.ticker = time.NewTicker(1 * time.Minute)
	s.running = true

	go s.run(ctx)

	return nil
}

// Stop stops the scheduler
func (s *SchedulerService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping scheduler service")
	close(s.stopChan)
	s.ticker.Stop()
	s.running = false

	return nil
}

// run is the main scheduler loop
func (s *SchedulerService) run(ctx context.Context) {
	// Run initial check
	s.checkSchedules(ctx)

	for {
		select {
		case <-s.ticker.C:
			s.checkSchedules(ctx)
		case <-s.stopChan:
			s.logger.Info("Scheduler stopped")
			return
		case <-ctx.Done():
			s.logger.Info("Scheduler context cancelled")
			return
		}
	}
}

// checkSchedules checks and executes due schedules
func (s *SchedulerService) checkSchedules(ctx context.Context) {
	schedules, err := s.repository.GetEnabled(ctx)
	if err != nil {
		s.logger.Error("Failed to get enabled schedules", "error", err)
		return
	}

	now := time.Now()
	currentDay := int(now.Weekday())
	currentHour := now.Hour()
	currentMinute := now.Minute()
	zone, offset := now.Zone()

	s.logger.Info("Checking schedules",
		"current_time", now.Format("2006-01-02 15:04:05"),
		"timezone", zone,
		"offset_seconds", offset,
		"day", currentDay,
		"hour", currentHour,
		"minute", currentMinute,
		"enabled_schedules", len(schedules))

	for _, schedule := range schedules {
		var shouldExecute bool
		var scheduleInfo string

		switch schedule.ScheduleType {
		case "once":
			// One-time schedule: check specific date and time
			if schedule.SpecificDate != nil {
				currentDate := now.Format("2006-01-02")
				scheduleDate := schedule.SpecificDate.Format("2006-01-02")

				shouldExecute = currentDate == scheduleDate &&
					schedule.Hour == currentHour &&
					schedule.Minute == currentMinute

				scheduleInfo = fmt.Sprintf("once=%s %02d:%02d", scheduleDate, schedule.Hour, schedule.Minute)
			}

		case "yearly":
			// Yearly recurring: check month, day, and time
			if schedule.Month != nil && schedule.DayOfMonth != nil {
				currentMonth := int(now.Month())
				currentDayOfMonth := now.Day()

				shouldExecute = *schedule.Month == currentMonth &&
					*schedule.DayOfMonth == currentDayOfMonth &&
					schedule.Hour == currentHour &&
					schedule.Minute == currentMinute

				scheduleInfo = fmt.Sprintf("yearly=%02d/%02d %02d:%02d", *schedule.Month, *schedule.DayOfMonth, schedule.Hour, schedule.Minute)
			}

		case "weekly":
			// Weekly recurring: check day of week and time
			if schedule.DayOfWeek != nil {
				shouldExecute = *schedule.DayOfWeek == currentDay &&
					schedule.Hour == currentHour &&
					schedule.Minute == currentMinute

				scheduleInfo = fmt.Sprintf("weekly=day_%d %02d:%02d", *schedule.DayOfWeek, schedule.Hour, schedule.Minute)

				s.logger.Info("Weekly schedule check",
					"name", schedule.Name,
					"schedule_day", *schedule.DayOfWeek,
					"schedule_time", fmt.Sprintf("%02d:%02d", schedule.Hour, schedule.Minute),
					"current_day", currentDay,
					"current_time", fmt.Sprintf("%02d:%02d", currentHour, currentMinute),
					"match", shouldExecute)
			}
		}

		s.logger.Debug("Checking schedule",
			"name", schedule.Name,
			"type", scheduleInfo,
			"matches", shouldExecute)

		if shouldExecute {
			// Check if already run in the last minute
			if schedule.LastRun != nil && now.Sub(*schedule.LastRun) < 1*time.Minute {
				s.logger.Debug("Schedule already run recently", "name", schedule.Name, "last_run", schedule.LastRun)
				continue
			}

			s.logger.Info("Executing schedule",
				"id", schedule.ID,
				"name", schedule.Name,
				"type", scheduleInfo,
				"group", schedule.GroupJID)

			// Execute in goroutine to avoid blocking
			go s.executeSchedule(ctx, schedule)

			// For one-time schedules, disable after execution
			if schedule.ScheduleType == "once" {
				go func(schedID string) {
					schedule.Enabled = false
					if err := s.repository.Update(ctx, schedule); err != nil {
						s.logger.Error("Failed to disable one-time schedule", "error", err, "schedule_id", schedID)
					}
				}(schedule.ID)
			}
		}
	}
}

// executeSchedule executes a single schedule
func (s *SchedulerService) executeSchedule(ctx context.Context, schedule *domain.Schedule) {
	execution := &domain.ScheduleExecution{
		ID:         uuid.New().String(),
		ScheduleID: schedule.ID,
		ExecutedAt: time.Now(),
	}

	// Update last run time
	if err := s.repository.UpdateLastRun(ctx, schedule.ID, execution.ExecutedAt); err != nil {
		s.logger.Error("Failed to update last run", "error", err, "schedule_id", schedule.ID)
	}

	// Call webhook with empty message (webhook can return scheduled content)
	response, err := s.webhookClient.Call(ctx, schedule.WebhookURL, "")
	if err != nil {
		s.logger.Error("Failed to call webhook for schedule",
			"error", err,
			"schedule_id", schedule.ID,
			"webhook_url", schedule.WebhookURL)

		execution.Success = false
		execution.Error = err.Error()
		s.repository.LogExecution(ctx, execution)
		return
	}

	// Handle response based on content type
	var responseContent string
	if response.ContentType == "image/jpeg" || response.ContentType == "image/png" {
		// Send as image
		s.logger.Info("Sending scheduled image",
			"size", len(response.Content),
			"mime", response.ContentType,
			"group", schedule.GroupJID)

		if err := s.whatsapp.SendImage(ctx, schedule.GroupJID, response.Content, response.ContentType, "", "", ""); err != nil {
			s.logger.Error("Failed to send scheduled image", "error", err)
			execution.Success = false
			execution.Error = fmt.Sprintf("failed to send image: %v", err)
			s.repository.LogExecution(ctx, execution)
			return
		}
		responseContent = "[Image sent]"
	} else {
		// Format and send as text
		formattedText := FormatWebhookResponse(response.TextContent)

		if err := s.whatsapp.SendMessage(ctx, schedule.GroupJID, formattedText); err != nil {
			s.logger.Error("Failed to send scheduled message", "error", err)
			execution.Success = false
			execution.Error = fmt.Sprintf("failed to send message: %v", err)
			s.repository.LogExecution(ctx, execution)
			return
		}
		responseContent = formattedText
	}

	// Log successful execution
	execution.Success = true
	execution.Response = responseContent
	if err := s.repository.LogExecution(ctx, execution); err != nil {
		s.logger.Error("Failed to log execution", "error", err)
	}

	s.logger.Info("Schedule executed successfully",
		"schedule_id", schedule.ID,
		"name", schedule.Name)
}

// CreateSchedule creates a new schedule
func (s *SchedulerService) CreateSchedule(ctx context.Context, schedule *domain.Schedule) error {
	schedule.ID = uuid.New().String()
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	if err := s.repository.Create(ctx, schedule); err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	s.logger.Info("Schedule created", "id", schedule.ID, "name", schedule.Name)
	return nil
}

// UpdateSchedule updates an existing schedule
func (s *SchedulerService) UpdateSchedule(ctx context.Context, schedule *domain.Schedule) error {
	if err := s.repository.Update(ctx, schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	s.logger.Info("Schedule updated", "id", schedule.ID)
	return nil
}

// DeleteSchedule deletes a schedule
func (s *SchedulerService) DeleteSchedule(ctx context.Context, id string) error {
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	s.logger.Info("Schedule deleted", "id", id)
	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *SchedulerService) GetSchedule(ctx context.Context, id string) (*domain.Schedule, error) {
	return s.repository.GetByID(ctx, id)
}

// GetAllSchedules retrieves all schedules
func (s *SchedulerService) GetAllSchedules(ctx context.Context) ([]*domain.Schedule, error) {
	return s.repository.GetAll(ctx)
}

// GetScheduleExecutions retrieves execution logs for a schedule
func (s *SchedulerService) GetScheduleExecutions(ctx context.Context, scheduleID string, limit int) ([]*domain.ScheduleExecution, error) {
	return s.repository.GetExecutions(ctx, scheduleID, limit)
}

// ServerTimeInfo contains server time information
type ServerTimeInfo struct {
	CurrentTime  time.Time `json:"current_time"`
	TimeZone     string    `json:"timezone"`
	UnixTime     int64     `json:"unix_time"`
	DayOfWeek    int       `json:"day_of_week"`
	Hour         int       `json:"hour"`
	Minute       int       `json:"minute"`
	FormattedStr string    `json:"formatted_str"`
}

// GetServerTime returns the server's current time and timezone info
func (s *SchedulerService) GetServerTime() *ServerTimeInfo {
	now := time.Now()
	zone, _ := now.Zone()

	return &ServerTimeInfo{
		CurrentTime:  now,
		TimeZone:     zone,
		UnixTime:     now.Unix(),
		DayOfWeek:    int(now.Weekday()),
		Hour:         now.Hour(),
		Minute:       now.Minute(),
		FormattedStr: now.Format("2006-01-02 15:04:05 MST"),
	}
}
