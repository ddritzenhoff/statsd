package stats

import "time"

// Date represents the month and year in which the slack reactions are grouped.
type Date struct {
	Month time.Month `json:"month"`
	Year  int64      `json:"year"`
}

// Member represents reactions pertaining to a particular member of the slack organization within a given month and year.
type Member struct {
	ID                int64  `json:"id"`
	SlackUID          string `json:"slackUID"`
	ReceivedLikes     int64  `json:"receivedLikes"`
	ReceivedDislikes  int64  `json:"receivedDislikes"`
	ReceivedReactions int64  `json:"receivedReactions"`
	GivenLikes        int64  `json:"givenLikes"`
	GivenDislikes     int64  `json:"givenDislikes"`
	GivenReactions    int64  `json:"givenReactions"`
	Date              Date   `json:"date"`
}

// MemberService represents a service for managing a Member.
type MemberService interface {
	// FindMemberByID retrieves a Member by ID.
	// Returns ErrNotFound if the ID does not exist.
	FindMemberByID(id int64) (*Member, error)

	// FindMember retrives a Member by his Slack User ID, the Month, and the Year.
	// Returns ErrNotFound if not matches found.
	FindMember(SlackUID string, Month time.Month, Year int64) (*Member, error)

	// CreateMember creates a new Member.
	CreateMember(m *Member) error

	// UpdateMember updates a Member.
	UpdateMember(id int64, upd MemberUpdate) error

	// Summary returns a summary of the members.
	Summary() (*MemberSummary, error)

	// DeleteMember permanently deletes a Member
	DeleteMember(id int64) error
}

type MemberSummary struct {
	MostLikesGiven        []Member
	MostDislikesGiven     []Member
	MostReactionsGiven    []Member
	MostLikesReceived     []Member
	MostDislikesReceived  []Member
	MostReactionsReceived []Member
}

// MemberUpdate represents a set of fields to be updated via UpdateMember().
type MemberUpdate struct {
	ReceivedLikes     *int64
	ReceivedDislikes  *int64
	ReceivedReactions *int64
	GivenLikes        *int64
	GivenDislikes     *int64
	GivenReactions    *int64
}
