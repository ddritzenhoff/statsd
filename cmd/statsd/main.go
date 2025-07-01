package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/ddritzenhoff/statsd/http"
	"github.com/ddritzenhoff/statsd/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DSN      string = "/data/statsd.db"
	HTTPAddr string = "0.0.0.0:8080"
)

// main is the entry point to the application binary.
func main() {
	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	m := &Main{}

	// Execute program.
	if err := m.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Wait for CTRL-C.
	<-ctx.Done()

	// clean up program
	m.Close()
}

// Main represents the program.
type Main struct {
	// SQLite database used by SQLite service implementations.
	DB *sqlite.DB

	// HTTP server for handling HTTP communication.
	// SQLite services are attached to it before running.
	HTTPServer *http.Server
}

// Run initializes the member and Slack services and starts the HTTP server.
func (m *Main) Run(ctx context.Context) error {
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	botSigningKey := os.Getenv("SLACK_BOT_SIGNING_KEY")

	m.DB = sqlite.NewDB(DSN)
	if err := m.DB.Open(); err != nil {
		return fmt.Errorf("db open: %w", err)
	}

	memberService := sqlite.NewMemberService(m.DB)
	leaderboardService := sqlite.NewLeaderboardService(m.DB)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slackService, err := http.NewSlackService(logger, memberService, leaderboardService, signingSecret, botSigningKey)
	if err != nil {
		return fmt.Errorf("Run NewSlackService: %w", err)
	}

	m.HTTPServer = http.NewServer(logger, HTTPAddr, slackService)
	if err := m.HTTPServer.Open(); err != nil {
		return fmt.Errorf("Run: %w", err)
	}

	return nil
}

// Close gracefully closes open http server and database connections.
func (m *Main) Close() error {
	if m.HTTPServer != nil {
		if err := m.HTTPServer.Close(); err != nil {
			return err
		}
	}
	if m.DB != nil {
		if err := m.DB.Close(); err != nil {
			return err
		}
	}
	return nil
}
