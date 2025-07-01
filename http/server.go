package http

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// ShutdownTimeout is the time given for outstanding requests to finish before shutdown.
const ShutdownTimeout = 1 * time.Second

// Server represents an HTTP server. It is meant to wrap all http functionality
// used by the application so that dependent packages (such as cmd/statsd) do
// not need to reference the "net/http" package at all.
type Server struct {
	ln     net.Listener
	server *http.Server
	router chi.Router

	// Dependencies
	addr         string
	slackService Slacker
	logger       *slog.Logger
}

// NewServer creates a new instance of Server.
func NewServer(logger *slog.Logger, serverAddr string, ss Slacker) *Server {
	s := &Server{
		server: &http.Server{},
		router: chi.NewRouter(),
	}
	// inject dependences
	s.addr = serverAddr
	s.slackService = ss
	s.logger = logger

	// create routes and attach handlers
	s.server.Handler = http.HandlerFunc(s.router.ServeHTTP)
	s.router.NotFound(s.handleNotFound)
	s.router.Get("/ping", s.handlePing)
	s.router.Post("/events", s.handleEvents)
	s.router.Route("/slack/", func(r chi.Router) {
		r.Post("/monthly-update", s.handleMonthlyUpdate)
	})
	return s
}

// Open establishes a connection to an address and begins listening for requests.
func (s *Server) Open() (err error) {
	// Open a listener on the bind address
	if s.ln, err = net.Listen("tcp", s.addr); err != nil {
		return fmt.Errorf("Open: %w", err)
	}
	s.logger.Info("server listening", slog.String("address", s.addr))
	go s.server.Serve(s.ln)
	return nil
}

// Close gracefully shuts down the server.
func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// handleMonthlyUpdate generates and monthly slack summary and publishes it.
func (s *Server) handleMonthlyUpdate(w http.ResponseWriter, r *http.Request) {
	err := s.slackService.HandleMonthlyUpdate(w, r)
	if err != nil {
		s.logger.Error(err.Error())
	}
	w.WriteHeader(http.StatusOK)
}

// handleEvents handles Slack push events.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	err := s.slackService.HandleEvents(w, r)
	if err != nil {
		s.logger.Error(err.Error())
	}
	w.WriteHeader(http.StatusOK)
}

// handlePing returns a basic 'pong' response when the server is pinged.
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong\n"))
}

// handleNotFound returns a basic 'not found' response when the requested resource doesn't exist.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Add("Content-Type", "text/plain")
	w.Write([]byte("Sorry, it looks like we couldn't find what you were looking for."))
}
