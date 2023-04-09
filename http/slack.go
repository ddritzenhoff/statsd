package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ddritzenhoff/stats"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

const (
	ThumbsUp   = "thumbsup"
	ThumbsDown = "thumbsdown"
)

// Slacker represents a service for handling Slack push events.
type Slacker interface {
	HandleEvents(w http.ResponseWriter, r *http.Request) error
	HandleMonthlyUpdate(w http.ResponseWriter, r *http.Request) error
}

// Slack represents a service for handling specific Slack events.
type Slack struct {
	// Services used by Slack
	MemberService stats.MemberService
	client        *slack.Client

	// Dependencies
	SigningSecret string
	ChannelID     string
}

// NewSlackService creates a new instance of slackService.
func NewSlackService(ms stats.MemberService, signingSecret string, botSigningKey string, channelID string) (Slacker, error) {
	return &Slack{
		MemberService: ms,
		client:        slack.New(botSigningKey),
		SigningSecret: signingSecret,
		ChannelID:     channelID,
	}, nil
}

func (s *Slack) HandleMonthlyUpdate(w http.ResponseWriter, r *http.Request) error {
	summary, err := s.MemberService.Summary()
	if err != nil {
		return fmt.Errorf("HandleMonthlyUpdate: %w", err)
	}

	var sectionBlocks []slack.Block
	headerText := slack.NewTextBlockObject("mrkdwn", "*Monthly Stats Update*", false, false)
	headerSection := slack.NewHeaderBlock(headerText)
	sectionBlocks = append(sectionBlocks, headerSection)

	mostLikesGivenMembers := "Most likes given this month: "
	for _, member := range summary.MostLikesGiven {
		mostLikesGivenMembers += fmt.Sprintf("<@%s> (%d)", member.SlackUID, member.GivenLikes)
	}
	sectionText := slack.NewTextBlockObject("mrkdwn", mostLikesGivenMembers, false, false)
	sectionBlocks = append(sectionBlocks, slack.NewSectionBlock(sectionText, nil, nil))

	mostLikesReceivedMembers := "Most likes received this month (aka good boy of the month): "
	for _, member := range summary.MostLikesReceived {
		mostLikesReceivedMembers += fmt.Sprintf("<@%s> (%d)", member.SlackUID, member.ReceivedLikes)
	}
	sectionText = slack.NewTextBlockObject("mrkdwn", mostLikesReceivedMembers, false, false)
	sectionBlocks = append(sectionBlocks, slack.NewSectionBlock(sectionText, nil, nil))

	mostDislikesGivenMembers := "Most dislikes given this month (aka most negative): "
	for _, member := range summary.MostDislikesGiven {
		mostDislikesGivenMembers += fmt.Sprintf("<@%s> (%d)", member.SlackUID, member.GivenDislikes)
	}
	sectionText = slack.NewTextBlockObject("mrkdwn", mostDislikesGivenMembers, false, false)
	sectionBlocks = append(sectionBlocks, slack.NewSectionBlock(sectionText, nil, nil))

	mostDislikesReceivedMembers := "Most dislikes received this month (aka the bad bad no good boys): "
	for _, member := range summary.MostDislikesReceived {
		mostDislikesReceivedMembers += fmt.Sprintf("<@%s> (%d)", member.SlackUID, member.ReceivedDislikes)
	}
	sectionText = slack.NewTextBlockObject("mrkdwn", mostDislikesReceivedMembers, false, false)
	sectionBlocks = append(sectionBlocks, slack.NewSectionBlock(sectionText, nil, nil))

	mostReactionsGiven := "Most reactions given this month (aka chill BRUH): "
	for _, member := range summary.MostReactionsGiven {
		mostReactionsGiven += fmt.Sprintf("<@%s> (%d)", member.SlackUID, member.GivenReactions)
	}
	sectionText = slack.NewTextBlockObject("mrkdwn", mostReactionsGiven, false, false)
	sectionBlocks = append(sectionBlocks, slack.NewSectionBlock(sectionText, nil, nil))

	mostReactionsReceived := "Most reactions received this month (aka Mr.Worldwide): "
	for _, member := range summary.MostReactionsReceived {
		mostReactionsReceived += fmt.Sprintf("<@%s> (%d)", member.SlackUID, member.ReceivedReactions)
	}
	sectionText = slack.NewTextBlockObject("mrkdwn", mostReactionsReceived, false, false)
	sectionBlocks = append(sectionBlocks, slack.NewSectionBlock(sectionText, nil, nil))
	compiledMsg := slack.MsgOptionBlocks(sectionBlocks...)
	_, _, err = s.client.PostMessage(s.ChannelID, compiledMsg)
	if err != nil {
		return fmt.Errorf("WeeklyUpdate PostMessage: %w", err)
	}
	return nil
}

// handleEvents handles Slack push events.
func (s *Slack) HandleEvents(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("HandleEvents: %w", err)
	}
	sv, err := slack.NewSecretsVerifier(r.Header, s.SigningSecret)
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
				w.WriteHeader(http.StatusInternalServerError)
				return fmt.Errorf("HandleEvents: %w", err)
			}
		case *slackevents.ReactionRemovedEvent:
			err := s.HandleReactionRemovedEvent(ev)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return fmt.Errorf("HandleEvents: %w", err)
			}
		}
	}
	return nil
}

// HandleReactionAddedEvent handles the event when a user reacts to the post of another user.
func (s *Slack) HandleReactionAddedEvent(e *slackevents.ReactionAddedEvent) error {
	year, month, _ := time.Now().Date()

	// If the user is reacting to his own message, do nothing.
	if e.User == e.ItemUser {
		return nil
	}

	// Create the member (user reacting to the material) if he does not already exist within the database.
	member, err := s.MemberService.FindMember(e.User, month, int64(year))
	if err != nil {
		if errors.Is(err, stats.ErrNotFound) {
			err = s.MemberService.CreateMember(&stats.Member{
				SlackUID: e.User,
				Date: stats.Date{
					Month: month,
					Year:  int64(year),
				},
			})
			if err != nil {
				return fmt.Errorf("HandleReactionAddedEvent CreateMember member: %w", err)
			}
			s.HandleReactionAddedEvent(e)
		}
		return fmt.Errorf("HandleReactionAddedEvent FindMember User: %w", err)
	}

	// Create the member (user being reacted to) if he does not already exist within the database.
	itemMember, err := s.MemberService.FindMember(e.ItemUser, month, int64(year))
	if err != nil {
		if errors.Is(err, stats.ErrNotFound) {
			err = s.MemberService.CreateMember(&stats.Member{
				SlackUID: e.User,
				Date: stats.Date{
					Month: month,
					Year:  int64(year),
				},
			})
			if err != nil {
				return fmt.Errorf("HandleReactionAddedEvent CreateMember itemMember: %w", err)
			}
			s.HandleReactionAddedEvent(e)
		}
		return fmt.Errorf("HandleReactionAddedEvent FindMember ItemUser: %w", err)
	}

	// Update the reactions.
	member.GivenReactions += 1
	itemMember.ReceivedReactions += 1
	if e.Reaction == ThumbsUp {
		member.GivenLikes += 1
		itemMember.ReceivedLikes += 1
	} else if e.Reaction == ThumbsDown {
		member.GivenDislikes += 1
		itemMember.ReceivedDislikes += 1
	}

	// Update the stats of the User reacting to the message.
	s.MemberService.UpdateMember(member.ID, stats.MemberUpdate{
		GivenReactions: &member.GivenReactions,
		GivenLikes:     &member.GivenLikes,
		GivenDislikes:  &member.GivenDislikes,
	})

	// Update the stats of the User being reacted to.
	s.MemberService.UpdateMember(itemMember.ID, stats.MemberUpdate{
		ReceivedReactions: &itemMember.ReceivedReactions,
		ReceivedLikes:     &itemMember.ReceivedLikes,
		ReceivedDislikes:  &itemMember.ReceivedDislikes,
	})
	return nil
}

// max finds the max between two int64s and returns it.
func max(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// HandleReactionRemovedEvent handles the event when a user removes a reaction from another user's post.
func (s *Slack) HandleReactionRemovedEvent(e *slackevents.ReactionRemovedEvent) error {
	year, month, _ := time.Now().Date()

	// If the user is reacting to his own message, do nothing.
	if e.User == e.ItemUser {
		return nil
	}

	// Create the member (user reacting to the material) if he does not already exist within the database.
	member, err := s.MemberService.FindMember(e.User, month, int64(year))
	if err != nil {
		if errors.Is(err, stats.ErrNotFound) {
			err = s.MemberService.CreateMember(&stats.Member{
				SlackUID: e.User,
				Date: stats.Date{
					Month: month,
					Year:  int64(year),
				},
			})
			if err != nil {
				return fmt.Errorf("HandleReactionRemovedEvent CreateMember member: %w", err)
			}
			s.HandleReactionRemovedEvent(e)
		}
		return fmt.Errorf("HandleReactionRemovedEvent FindMember User: %w", err)
	}

	// Create the member (user being reacted to) if he does not already exist within the database.
	itemMember, err := s.MemberService.FindMember(e.ItemUser, month, int64(year))
	if err != nil {
		if errors.Is(err, stats.ErrNotFound) {
			err = s.MemberService.CreateMember(&stats.Member{
				SlackUID: e.User,
				Date: stats.Date{
					Month: month,
					Year:  int64(year),
				},
			})
			if err != nil {
				return fmt.Errorf("HandleReactionRemovedEvent CreateMember itemMember: %w", err)
			}
			s.HandleReactionRemovedEvent(e)
		}
		return fmt.Errorf("HandleReactionRemovedEvent FindMember ItemUser: %w", err)
	}

	// Update the reactions.
	member.GivenReactions = max(member.GivenReactions-1, 0)
	itemMember.ReceivedReactions = max(itemMember.ReceivedReactions-1, 0)
	if e.Reaction == ThumbsUp {
		member.GivenLikes = max(member.GivenLikes-1, 0)
		itemMember.ReceivedLikes = max(itemMember.ReceivedLikes-1, 0)
	} else if e.Reaction == ThumbsDown {
		member.GivenDislikes = max(member.GivenDislikes-1, 0)
		itemMember.ReceivedDislikes = max(itemMember.ReceivedDislikes-1, 0)
	}

	// Update the stats of the User reacting to the message.
	s.MemberService.UpdateMember(member.ID, stats.MemberUpdate{
		GivenReactions: &member.GivenReactions,
		GivenLikes:     &member.GivenLikes,
		GivenDislikes:  &member.GivenDislikes,
	})

	// Update the stats of the User being reacted to.
	s.MemberService.UpdateMember(itemMember.ID, stats.MemberUpdate{
		ReceivedReactions: &itemMember.ReceivedReactions,
		ReceivedLikes:     &itemMember.ReceivedLikes,
		ReceivedDislikes:  &itemMember.ReceivedDislikes,
	})
	return nil
}
