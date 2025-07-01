package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ddritzenhoff/statsd"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

const (
	ThumbsUp   = "+1"
	ThumbsDown = "-1"
)

// Slacker represents a service for handling Slack push events.
type Slacker interface {
	HandleEvents(w http.ResponseWriter, r *http.Request) error
	HandleMonthlyUpdate(w http.ResponseWriter, r *http.Request) error
}

// Slack represents a service for handling specific Slack events.
type Slack struct {
	// Services used by Slack
	LeaderboardService statsd.LeaderboardService
	MemberService      statsd.MemberService
	client             *slack.Client

	// Dependencies
	logger        *slog.Logger
	signingSecret string
}

// NewSlackService creates a new instance of slackService.
func NewSlackService(logger *slog.Logger, ms statsd.MemberService, ls statsd.LeaderboardService, signingSecret string, botSigningKey string) (Slacker, error) {
	return &Slack{
		logger:             logger,
		MemberService:      ms,
		LeaderboardService: ls,
		client:             slack.New(botSigningKey),
		signingSecret:      signingSecret,
	}, nil
}

// HandleMonthlyUpdate sends a summary of the recorded metrics into Slack.
//
// Expecting x-www-form-urlencoded payload in the form of `channel=<channelID>&date=<month>-<year>`.
// I.e. to represent October 2023, the key=value combination would be `date=10-2023`.
func (s *Slack) HandleMonthlyUpdate(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	channelID := r.PostForm.Get("channel")
	if channelID == "" {
		return errors.New("no channel value provided within the form")
	}
	rawDate := r.PostForm.Get("date")
	if rawDate == "" {
		return errors.New("no date value provided within the form")
	}
	date, err := statsd.NewMonthYearString(rawDate)
	if err != nil {
		return err
	}

	leaderboard, err := s.LeaderboardService.FindLeaderboard(date)
	if err != nil {
		return err
	}

	month, err := date.Month()
	if err != nil {
		return err
	}

	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Slack member activity for the month of %s", month), false, false),
			nil,
			nil,
		),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("- most likes received: <@%s> with %d likes", leaderboard.MostReceivedLikesMember.SlackUID, leaderboard.MostReceivedLikesMember.ReceivedLikes), false, false),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("- hottest takes (most dislikes received): <@%s> with %d dislikes", leaderboard.MostReceivedDislikesMember.SlackUID, leaderboard.MostReceivedDislikesMember.ReceivedDislikes), false, false),
			nil,
			nil,
		),
	}

	msg := slack.NewBlockMessage(blocks...)

	_, _, err = s.client.PostMessage(channelID, slack.MsgOptionBlocks(msg.Blocks.BlockSet...))
	if err != nil {
		return fmt.Errorf("WeeklyUpdate PostMessage: %w", err)
	}

	s.logger.Info("published monthly update", slog.String("month", month))
	return nil
}

// handleEvents handles Slack push events.
func (s *Slack) HandleEvents(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("HandleEvents: %w", err)
	}
	sv, err := slack.NewSecretsVerifier(r.Header, s.signingSecret)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("HandleEvents: %w", err)
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("HandleEvents: %w", err)
	}
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return fmt.Errorf("HandleEvents: %w", err)
	}
	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("HandleEvents: %w", err)
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return fmt.Errorf("HandleEvents: %w", err)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(r.Challenge))
	}
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.ReactionAddedEvent:
			err := s.HandleReactionAddedEvent(ev)
			if err != nil {
				return fmt.Errorf("HandleEvents: %w", err)
			}
		case *slackevents.ReactionRemovedEvent:
			err := s.HandleReactionRemovedEvent(ev)
			if err != nil {
				return fmt.Errorf("HandleEvents: %w", err)
			}
		}
	}
	return nil
}

// HandleReactionEvent handles an event by updating the member with the specified slackUID.
func (s *Slack) HandleReactionEvent(memSlackUID string, update func(m *statsd.Member)) error {
	if memSlackUID == "USLACKBOT" || memSlackUID == "" {
		s.logger.Info("reaction to invalid target", slog.String("target slackUID", memSlackUID))
		return nil
	}
	monthYear := statsd.NewMonthYear(time.Now().UTC())

	// Create the member if he does not already exist within the database.
	mem, err := s.MemberService.FindMember(memSlackUID, monthYear)
	if errors.Is(err, statsd.ErrNotFound) {
		m := &statsd.Member{
			SlackUID: memSlackUID,
			Date:     monthYear,
		}
		err := s.MemberService.CreateMember(m)
		if err != nil {
			return fmt.Errorf("HandleReactionAddedEvent CreateMember itemMember: %w", err)
		}
		s.logger.Info("created new member", slog.String("slackUID", m.SlackUID), slog.String("date", monthYear.String()))
		mem = m
	} else if err != nil {
		return fmt.Errorf("HandleReactionAddedEvent FindMember ItemUser: %w", err)
	}

	update(mem)

	// Update the stats of the User being reacted to.
	m, err := s.MemberService.UpdateMember(mem.ID, statsd.MemberUpdate{
		ReceivedLikes:    &mem.ReceivedLikes,
		ReceivedDislikes: &mem.ReceivedDislikes,
	})
	if err != nil {
		return err
	}
	s.logger.Info("updated user", slog.String("slackUID", m.SlackUID), slog.Int("received likes", m.ReceivedLikes), slog.Int("received dislikes", m.ReceivedDislikes))
	return nil
}

// HandleReactionAddedEvent handles the event when a user reacts to the post of another user.
func (s *Slack) HandleReactionAddedEvent(e *slackevents.ReactionAddedEvent) error {
	switch e.Reaction {
	case ThumbsUp:
		return s.HandleReactionEvent(e.ItemUser, func(m *statsd.Member) {
			m.ReceivedLikes += 1
		})
	case ThumbsDown:
		return s.HandleReactionEvent(e.ItemUser, func(m *statsd.Member) {
			m.ReceivedDislikes += 1
		})
	}
	return nil
}

// HandleReactionRemovedEvent handles the event when a user removes a reaction from another user's post.
func (s *Slack) HandleReactionRemovedEvent(e *slackevents.ReactionRemovedEvent) error {
	switch e.Reaction {
	case ThumbsUp:
		return s.HandleReactionEvent(e.ItemUser, func(m *statsd.Member) {
			m.ReceivedLikes = max(m.ReceivedLikes-1, 0)
		})
	case ThumbsDown:
		return s.HandleReactionEvent(e.ItemUser, func(m *statsd.Member) {
			m.ReceivedDislikes = max(m.ReceivedDislikes-1, 0)
		})
	}
	return nil
}
