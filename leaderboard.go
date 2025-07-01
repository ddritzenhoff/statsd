package statsd

// Leaderboard represents the Slack user(s) with the most likes and dislikes for a particular month in a given year.
type Leaderboard struct {
	Date                       MonthYear
	MostReceivedLikesMember    Member
	MostReceivedDislikesMember Member
}

// LeaderboardService represents a service for managing a Leaderboard.
type LeaderboardService interface {
	// FindLeaderboard retrives a Leadboard by its date (year and month).
	// Returns ErrNotFound if no matches are found.
	FindLeaderboard(Date MonthYear) (*Leaderboard, error)
}
