package statsd

import (
	"fmt"
	"time"
)

// MonthYear represents a string with the following format: <month>-<year>.
// I.e. `02-2024` represents February 2024.
type MonthYear string

const monthYearLayout string = "01-2006"

// NewMonthYear returns a new instance of MonthYear.
func NewMonthYear(t time.Time) MonthYear {
	return MonthYear(t.UTC().Format(monthYearLayout))
}

// NewMonthYearString returns a new instance of MonthYear.
func NewMonthYearString(s string) (MonthYear, error) {
	t, err := time.Parse(monthYearLayout, s)
	if err != nil {
		return "", err
	}
	return NewMonthYear(t), nil
}

// String returns the string representation of MonthYear.
func (my *MonthYear) String() string {
	return string(*my)
}

// Month returns the English name of the corresponding month.
func (my *MonthYear) Month() (string, error) {
	t, err := time.Parse(monthYearLayout, my.String())
	if err != nil {
		return "", fmt.Errorf("unable to parse the MonthYear: %s", my.String())
	}
	return t.Month().String(), nil
}

// Member represents reactions pertaining to a particular member of the slack organization within a given month and year.
type Member struct {
	ID               int       `json:"id"`
	Date             MonthYear `json:"date"`
	SlackUID         string    `json:"slackUID"`
	ReceivedLikes    int       `json:"receivedLikes"`
	ReceivedDislikes int       `json:"receivedDislikes"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `jons:"updatedAt"`
}

// Validate returns an error if the member contains invalid fields.
// This only performs basic validation.
func (m *Member) Validate() error {
	if m.SlackUID == "" {
		return fmt.Errorf("slack user ID required %w", ErrInvalid)
	}
	return nil
}

// MemberService represents a service for managing a Member.
type MemberService interface {
	// FindMemberByID retrieves a Member by ID.
	// Returns ErrNotFound if the ID does not exist.
	FindMemberByID(id int) (*Member, error)

	// FindMember retrives a Member by his Slack User ID, and date (month and year).
	// Returns ErrNotFound if no matches found.
	FindMember(SlackUID string, date MonthYear) (*Member, error)

	// CreateMember creates a new Member.
	CreateMember(m *Member) error

	// UpdateMember updates a Member.
	// Returns ErrNotFound if the member does not exist.
	UpdateMember(id int, upd MemberUpdate) (*Member, error)

	// DeleteMember permanently deletes a Member
	DeleteMember(id int) error
}

// MemberUpdate represents a set of fields to be updated via UpdateMember().
type MemberUpdate struct {
	ReceivedLikes    *int
	ReceivedDislikes *int
}
