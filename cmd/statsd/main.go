package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/ddritzenhoff/stats/http"
	"github.com/ddritzenhoff/stats/sqlite"
	"github.com/ddritzenhoff/stats/sqlite/gen"
	_ "github.com/mattn/go-sqlite3"
	"github.com/peterbourgon/ff/v3"
)

func main() {
	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	// Execute program.
	if err := Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Wait for CTRL-C.
	<-ctx.Done()
}

// Run initializes the member and Slack services and starts the HTTP server.
func Run(ctx context.Context) error {
	// setup flags
	fs := flag.NewFlagSet("statsd", flag.ContinueOnError)
	var (
		listenAddr         = fs.String("listen-addr", "localhost:8080", "listen address")
		dsn                = fs.String("dsn", "~/programming/databases/stats.db", "database connection string")
		slackSigningSecret = fs.String("signing-secret", "", "to verify Slack requests")
		slackBotSigningKey = fs.String("bot-signing-key", "", "to send messages into the Slack workspace")
		slackChannelID     = fs.String("channel-id", "", "to send messages into a specific channel")
		_                  = fs.String("config", "", "config file (extension: .json)")
	)

	err := ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.JSONParser),
	)
	if err != nil {
		return fmt.Errorf("Run ff.Parse: %w", err)
	}
	DSNPath, err := expandDSN(*dsn)
	if err != nil {
		return fmt.Errorf("Run expandDSN: %w", err)
	}
	db, err := sqlite.Open(DSNPath)
	if err != nil {
		return fmt.Errorf("Run sqlite.Open: %w", err)
	}

	queries := gen.New(db)

	memberService := sqlite.NewMemberService(queries, db)

	slackService, err := http.NewSlackService(memberService, *slackSigningSecret, *slackBotSigningKey, *slackChannelID)
	if err != nil {
		return fmt.Errorf("Run NewSlackService: %w", err)
	}
	logger := log.New(os.Stdout, "statsd ", log.LstdFlags)
	httpServer := http.NewServer(logger, *listenAddr, slackService)
	if err := httpServer.Open(); err != nil {
		return fmt.Errorf("Run: %w", err)
	}
	return nil
}

// expand returns path using tilde expansion. This means that a file path that
// begins with the "~" will be expanded to prefix the user's home directory.
func expand(path string) (string, error) {
	// Ignore if path has no leading tilde.
	if path != "~" && !strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		return path, nil
	}

	// Fetch the current user to determine the home path.
	u, err := user.Current()
	if err != nil {
		return path, fmt.Errorf("expand user.Current: %w", err)
	} else if u.HomeDir == "" {
		return path, fmt.Errorf("expand u.HomeDir: home directory unset")
	}

	if path == "~" {
		return u.HomeDir, nil
	}
	return filepath.Join(u.HomeDir, strings.TrimPrefix(path, "~"+string(os.PathSeparator))), nil
}

// expandDSN expands a datasource name. Ignores in-memory databases.
func expandDSN(dsn string) (string, error) {
	if dsn == ":memory:" {
		return dsn, nil
	}
	return expand(dsn)
}
