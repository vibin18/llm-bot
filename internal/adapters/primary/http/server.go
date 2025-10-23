package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Server represents the HTTP server
type Server struct {
	server           *http.Server
	handlers         *Handlers
	scheduleHandlers *ScheduleHandlers
	logger           *slog.Logger
}

// NewServer creates a new HTTP server
func NewServer(port int, handlers *Handlers, scheduleHandlers *ScheduleHandlers, logger *slog.Logger) *Server {
	return &Server{
		handlers:         handlers,
		scheduleHandlers: scheduleHandlers,
		logger:           logger,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/groups", s.handlers.GetGroups).Methods("GET")
	api.HandleFunc("/config/allowed-groups", s.handlers.GetAllowedGroups).Methods("GET")
	api.HandleFunc("/config/allowed-groups", s.handlers.UpdateAllowedGroups).Methods("POST")
	api.HandleFunc("/webhooks", s.handlers.GetWebhooks).Methods("GET")
	api.HandleFunc("/webhooks", s.handlers.AddWebhook).Methods("POST")
	api.HandleFunc("/webhooks", s.handlers.DeleteWebhook).Methods("DELETE")
	api.HandleFunc("/status", s.handlers.GetStatus).Methods("GET")
	api.HandleFunc("/auth/qr", s.handlers.GetQRCode).Methods("GET")
	api.HandleFunc("/health", s.handlers.HealthCheck).Methods("GET")

	// Schedule routes
	if s.scheduleHandlers != nil {
		api.HandleFunc("/schedules", s.scheduleHandlers.GetSchedules).Methods("GET")
		api.HandleFunc("/schedules", s.scheduleHandlers.CreateSchedule).Methods("POST")
		api.HandleFunc("/schedules/{id}", s.scheduleHandlers.GetSchedule).Methods("GET")
		api.HandleFunc("/schedules/{id}", s.scheduleHandlers.UpdateSchedule).Methods("PUT")
		api.HandleFunc("/schedules/{id}", s.scheduleHandlers.DeleteSchedule).Methods("DELETE")
		api.HandleFunc("/schedules/{id}/executions", s.scheduleHandlers.GetScheduleExecutions).Methods("GET")
		api.HandleFunc("/server-time", s.scheduleHandlers.GetServerTime).Methods("GET")
	}

	// Static files and admin UI
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	router.HandleFunc("/schedules", s.serveSchedulesUI).Methods("GET")
	router.HandleFunc("/execution-logs", s.serveExecutionLogsUI).Methods("GET")
	router.HandleFunc("/groups", s.serveGroupsUI).Methods("GET")
	router.HandleFunc("/webhooks", s.serveWebhooksUI).Methods("GET")
	router.HandleFunc("/", s.serveAdminUI).Methods("GET")

	// Add CORS middleware
	router.Use(corsMiddleware)

	// Add logging middleware
	router.Use(s.loggingMiddleware)

	s.server.Handler = router

	s.logger.Info("Starting HTTP server", "addr", s.server.Addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}

// serveAdminUI serves the admin UI page
func (s *Server) serveAdminUI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/admin_new.html")
}

// serveSchedulesUI serves the schedules UI page
func (s *Server) serveSchedulesUI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/schedules_new.html")
}

// serveExecutionLogsUI serves the execution logs UI page
func (s *Server) serveExecutionLogsUI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/execution_logs_new.html")
}

// serveGroupsUI serves the groups management UI page
func (s *Server) serveGroupsUI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/groups_new.html")
}

// serveWebhooksUI serves the webhooks management UI page
func (s *Server) serveWebhooksUI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/webhooks_new.html")
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		s.logger.Debug("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start))
	})
}
