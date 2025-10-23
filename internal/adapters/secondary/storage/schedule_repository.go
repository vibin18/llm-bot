package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
	_ "github.com/mattn/go-sqlite3"
)

// ScheduleRepository implements domain.ScheduleRepository using SQLite
type ScheduleRepository struct {
	db *sql.DB
}

// NewScheduleRepository creates a new schedule repository
func NewScheduleRepository(dbPath string) (*ScheduleRepository, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &ScheduleRepository{db: db}
	if err := repo.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return repo, nil
}

// initialize creates the necessary tables
func (r *ScheduleRepository) initialize() error {
	schema := `
	CREATE TABLE IF NOT EXISTS schedules (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		group_jid TEXT NOT NULL,
		webhook_url TEXT NOT NULL,
		schedule_type TEXT NOT NULL DEFAULT 'weekly',
		day_of_week INTEGER,
		month INTEGER,
		day_of_month INTEGER,
		hour INTEGER NOT NULL,
		minute INTEGER NOT NULL,
		specific_date DATE,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		last_run DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS schedule_executions (
		id TEXT PRIMARY KEY,
		schedule_id TEXT NOT NULL,
		executed_at DATETIME NOT NULL,
		success BOOLEAN NOT NULL,
		error TEXT,
		response TEXT,
		FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled);
	CREATE INDEX IF NOT EXISTS idx_schedules_type ON schedules(schedule_type);
	CREATE INDEX IF NOT EXISTS idx_schedules_day_time ON schedules(day_of_week, hour, minute);
	CREATE INDEX IF NOT EXISTS idx_schedules_yearly ON schedules(month, day_of_month, hour, minute);
	CREATE INDEX IF NOT EXISTS idx_schedules_specific_date ON schedules(specific_date, hour, minute);
	CREATE INDEX IF NOT EXISTS idx_executions_schedule ON schedule_executions(schedule_id, executed_at DESC);
	`

	_, err := r.db.Exec(schema)
	return err
}

// Create creates a new schedule
func (r *ScheduleRepository) Create(ctx context.Context, schedule *domain.Schedule) error {
	query := `
		INSERT INTO schedules (id, name, group_jid, webhook_url, schedule_type, day_of_week, month, day_of_month, hour, minute, specific_date, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var specificDate *string
	if schedule.SpecificDate != nil {
		dateStr := schedule.SpecificDate.Format("2006-01-02")
		specificDate = &dateStr
	}

	_, err := r.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.Name,
		schedule.GroupJID,
		schedule.WebhookURL,
		schedule.ScheduleType,
		schedule.DayOfWeek,
		schedule.Month,
		schedule.DayOfMonth,
		schedule.Hour,
		schedule.Minute,
		specificDate,
		schedule.Enabled,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)

	return err
}

// GetByID retrieves a schedule by ID
func (r *ScheduleRepository) GetByID(ctx context.Context, id string) (*domain.Schedule, error) {
	query := `
		SELECT id, name, group_jid, webhook_url, schedule_type, day_of_week, month, day_of_month, hour, minute, specific_date, enabled, last_run, created_at, updated_at
		FROM schedules WHERE id = ?
	`

	schedule := &domain.Schedule{}
	var lastRun sql.NullTime
	var dayOfWeek, month, dayOfMonth sql.NullInt64
	var specificDate sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&schedule.ID,
		&schedule.Name,
		&schedule.GroupJID,
		&schedule.WebhookURL,
		&schedule.ScheduleType,
		&dayOfWeek,
		&month,
		&dayOfMonth,
		&schedule.Hour,
		&schedule.Minute,
		&specificDate,
		&schedule.Enabled,
		&lastRun,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found")
	}
	if err != nil {
		return nil, err
	}

	if dayOfWeek.Valid {
		day := int(dayOfWeek.Int64)
		schedule.DayOfWeek = &day
	}

	if month.Valid {
		m := int(month.Int64)
		schedule.Month = &m
	}

	if dayOfMonth.Valid {
		d := int(dayOfMonth.Int64)
		schedule.DayOfMonth = &d
	}

	if specificDate.Valid {
		parsed, err := time.Parse("2006-01-02", specificDate.String)
		if err == nil {
			schedule.SpecificDate = &parsed
		}
	}

	if lastRun.Valid {
		schedule.LastRun = &lastRun.Time
	}

	return schedule, nil
}

// GetAll retrieves all schedules
func (r *ScheduleRepository) GetAll(ctx context.Context) ([]*domain.Schedule, error) {
	query := `
		SELECT id, name, group_jid, webhook_url, schedule_type, day_of_week, month, day_of_month, hour, minute, specific_date, enabled, last_run, created_at, updated_at
		FROM schedules ORDER BY schedule_type, specific_date, month, day_of_month, day_of_week, hour, minute
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanSchedules(rows)
}

// GetEnabled retrieves all enabled schedules
func (r *ScheduleRepository) GetEnabled(ctx context.Context) ([]*domain.Schedule, error) {
	query := `
		SELECT id, name, group_jid, webhook_url, schedule_type, day_of_week, month, day_of_month, hour, minute, specific_date, enabled, last_run, created_at, updated_at
		FROM schedules WHERE enabled = 1 ORDER BY schedule_type, specific_date, month, day_of_month, day_of_week, hour, minute
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanSchedules(rows)
}

// Update updates an existing schedule
func (r *ScheduleRepository) Update(ctx context.Context, schedule *domain.Schedule) error {
	query := `
		UPDATE schedules
		SET name = ?, group_jid = ?, webhook_url = ?, schedule_type = ?, day_of_week = ?, month = ?, day_of_month = ?, hour = ?, minute = ?, specific_date = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`

	var specificDate *string
	if schedule.SpecificDate != nil {
		dateStr := schedule.SpecificDate.Format("2006-01-02")
		specificDate = &dateStr
	}

	_, err := r.db.ExecContext(ctx, query,
		schedule.Name,
		schedule.GroupJID,
		schedule.WebhookURL,
		schedule.ScheduleType,
		schedule.DayOfWeek,
		schedule.Month,
		schedule.DayOfMonth,
		schedule.Hour,
		schedule.Minute,
		specificDate,
		schedule.Enabled,
		time.Now(),
		schedule.ID,
	)

	return err
}

// Delete deletes a schedule
func (r *ScheduleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM schedules WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// UpdateLastRun updates the last run time of a schedule
func (r *ScheduleRepository) UpdateLastRun(ctx context.Context, id string, lastRun time.Time) error {
	query := `UPDATE schedules SET last_run = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, lastRun, id)
	return err
}

// LogExecution logs a schedule execution
func (r *ScheduleRepository) LogExecution(ctx context.Context, execution *domain.ScheduleExecution) error {
	query := `
		INSERT INTO schedule_executions (id, schedule_id, executed_at, success, error, response)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		execution.ID,
		execution.ScheduleID,
		execution.ExecutedAt,
		execution.Success,
		execution.Error,
		execution.Response,
	)

	return err
}

// GetExecutions retrieves execution logs for a schedule
func (r *ScheduleRepository) GetExecutions(ctx context.Context, scheduleID string, limit int) ([]*domain.ScheduleExecution, error) {
	query := `
		SELECT id, schedule_id, executed_at, success, error, response
		FROM schedule_executions WHERE schedule_id = ?
		ORDER BY executed_at DESC LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, scheduleID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []*domain.ScheduleExecution
	for rows.Next() {
		exec := &domain.ScheduleExecution{}
		var errorMsg, response sql.NullString

		err := rows.Scan(
			&exec.ID,
			&exec.ScheduleID,
			&exec.ExecutedAt,
			&exec.Success,
			&errorMsg,
			&response,
		)
		if err != nil {
			return nil, err
		}

		if errorMsg.Valid {
			exec.Error = errorMsg.String
		}
		if response.Valid {
			exec.Response = response.String
		}

		executions = append(executions, exec)
	}

	return executions, nil
}

// scanSchedules is a helper to scan multiple schedule rows
func (r *ScheduleRepository) scanSchedules(rows *sql.Rows) ([]*domain.Schedule, error) {
	var schedules []*domain.Schedule

	for rows.Next() {
		schedule := &domain.Schedule{}
		var lastRun sql.NullTime
		var dayOfWeek, month, dayOfMonth sql.NullInt64
		var specificDate sql.NullString

		err := rows.Scan(
			&schedule.ID,
			&schedule.Name,
			&schedule.GroupJID,
			&schedule.WebhookURL,
			&schedule.ScheduleType,
			&dayOfWeek,
			&month,
			&dayOfMonth,
			&schedule.Hour,
			&schedule.Minute,
			&specificDate,
			&schedule.Enabled,
			&lastRun,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if dayOfWeek.Valid {
			day := int(dayOfWeek.Int64)
			schedule.DayOfWeek = &day
		}

		if month.Valid {
			m := int(month.Int64)
			schedule.Month = &m
		}

		if dayOfMonth.Valid {
			d := int(dayOfMonth.Int64)
			schedule.DayOfMonth = &d
		}

		if specificDate.Valid {
			// Try multiple date formats
			var parsed time.Time
			var err error

			// Try with timestamp first (2025-10-23T00:00:00Z)
			parsed, err = time.Parse(time.RFC3339, specificDate.String)
			if err != nil {
				// Try date-only format (2025-10-23)
				parsed, err = time.Parse("2006-01-02", specificDate.String)
			}

			if err != nil {
				// Log parsing error but continue
				fmt.Printf("WARNING: Failed to parse specific_date '%s' for schedule %s: %v\n", specificDate.String, schedule.Name, err)
			} else {
				schedule.SpecificDate = &parsed
			}
		}

		if lastRun.Valid {
			schedule.LastRun = &lastRun.Time
		}

		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// Close closes the database connection
func (r *ScheduleRepository) Close() error {
	return r.db.Close()
}
