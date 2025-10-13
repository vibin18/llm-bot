package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
	"github.com/vibin/whatsapp-llm-bot/internal/core/services"
)

// ScheduleHandlers contains schedule-related HTTP handlers
type ScheduleHandlers struct {
	scheduler *services.SchedulerService
}

// NewScheduleHandlers creates new schedule handlers
func NewScheduleHandlers(scheduler *services.SchedulerService) *ScheduleHandlers {
	return &ScheduleHandlers{
		scheduler: scheduler,
	}
}

// GetSchedules returns all schedules
func (h *ScheduleHandlers) GetSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.scheduler.GetAllSchedules(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedules)
}

// GetSchedule returns a single schedule
func (h *ScheduleHandlers) GetSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	schedule, err := h.scheduler.GetSchedule(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

// CreateSchedule creates a new schedule
func (h *ScheduleHandlers) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var schedule domain.Schedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.scheduler.CreateSchedule(r.Context(), &schedule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(schedule)
}

// UpdateSchedule updates an existing schedule
func (h *ScheduleHandlers) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var schedule domain.Schedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	schedule.ID = id
	if err := h.scheduler.UpdateSchedule(r.Context(), &schedule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

// DeleteSchedule deletes a schedule
func (h *ScheduleHandlers) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.scheduler.DeleteSchedule(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetScheduleExecutions returns execution logs for a schedule
func (h *ScheduleHandlers) GetScheduleExecutions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	executions, err := h.scheduler.GetScheduleExecutions(r.Context(), id, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}
