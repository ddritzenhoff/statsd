package http

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Server represents an HTTP server. It is meant to wrap all http functionality
// used by the application so that dependent packages (such as cmd/statsd) do
// not need to reference the "net/http" package at all.
type Server struct {
	ln     net.Listener
	server *http.Server
	router chi.Router

	// Dependencies
	Addr         string
	SlackService Slacker
	logger       *log.Logger
}

// NewServer creates a new instance of Server.
func NewServer(logger *log.Logger, serverAddr string, ss Slacker) *Server {
	s := &Server{
		server: &http.Server{},
		router: chi.NewRouter(),
	}
	// inject dependences
	s.Addr = serverAddr
	s.SlackService = ss
	s.logger = logger

	// create routes and attach handlers
	s.server.Handler = http.HandlerFunc(s.router.ServeHTTP)
	s.router.NotFound(s.handleNotFound)
	s.router.Get("/ping", s.handlePing)
	s.router.Post("/events", s.handleEvents)
	s.router.Route("/slack/", func(r chi.Router) {
		r.Get("/monthly-update", s.handleMonthlyUpdate)
	})
	return s
}

// handleMonthlyUpdate generates and monthly slack summary and publishes it.
func (s *Server) handleMonthlyUpdate(w http.ResponseWriter, r *http.Request) {
	err := s.SlackService.HandleMonthlyUpdate(w, r)
	if err != nil {
		s.logger.Println(err)
	}
}

// Open establishes a connection to an address and begins listening for requests.
func (s *Server) Open() (err error) {
	// Open a listener on the bind address
	if s.ln, err = net.Listen("tcp", s.Addr); err != nil {
		return fmt.Errorf("Open: %w", err)
	}
	s.logger.Printf("Listening on %s", s.Addr)
	go s.server.Serve(s.ln)
	return nil
}

// handleEvents handles Slack push events.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	err := s.SlackService.HandleEvents(w, r)
	if err != nil {
		s.logger.Println(err)
	}
}

// handlePing returns a basic 'pong' response when the server is pinged.
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

// handleNotFound returns a basic 'not found' response when the requested resource doesn't exist.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Add("Content-Type", "text/plain")
	w.Write([]byte("Sorry, it looks like we couldn't find what you were looking for."))
}
