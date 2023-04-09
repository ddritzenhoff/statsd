package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	monthlyUpdate := &ffcli.Command{
		Name:       "monthly_update",
		ShortUsage: "stats monthly_update [address]",
		ShortHelp:  "Crunch all of the data for the current month and publish the results in Slack.",
		Exec: func(_ context.Context, args []string) error {
			if len(args) == 0 {
				return errors.New("must provide address")
			}
			_, err := http.Get(fmt.Sprintf("%s/slack/monthly-update", args[0]))
			if err != nil {
				return fmt.Errorf("error making request: %w", err)
			}
			return nil
		},
	}

	ping := &ffcli.Command{
		Name:       "ping",
		ShortUsage: "stats ping [address]",
		ShortHelp:  "Ping the server to see if it's up.",
		Exec: func(_ context.Context, args []string) error {
			if len(args) == 0 {
				return errors.New("must provide address")
			}
			resp, err := http.Get(fmt.Sprintf("%s/ping", args[0]))
			if err != nil {
				return fmt.Errorf("error making request: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("error reading response body: %w", err)
			}
			fmt.Println(string(body))
			return nil
		},
	}

	root := &ffcli.Command{
		ShortUsage:  "stats [flags] <subcommand>",
		Subcommands: []*ffcli.Command{monthlyUpdate, ping},
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
	}

	if err := root.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
