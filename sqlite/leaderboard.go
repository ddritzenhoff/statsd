package sqlite

import (
	"context"

	"github.com/ddritzenhoff/stats"
)

// Ensure service implements interface.
var _ stats.LeaderboardService = (*LeaderboardService)(nil)

// LeaderboardService represents a service for managing Members.
type LeaderboardService struct {
	db *DB
}

// NewLeaderboardService returns a new instance of MemberService.
func NewLeaderboardService(db *DB) *LeaderboardService {
	return &LeaderboardService{
		db: db,
	}
}

// FindLeaderboard retrives a Leadboard by its date (year and month).
// Returns ErrNotFound if no matches are found.
func (ls *LeaderboardService) FindLeaderboard(date stats.MonthYear) (*stats.Leaderboard, error) {
	genMostReceivedLikesMember, err := ls.db.query.MostLikesReceived(context.TODO(), date.String())
	if err != nil {
		return nil, err
	}
	mostReceivedLikesMember, err := genMemberToMember(&genMostReceivedLikesMember)
	if err != nil {
		return nil, err
	}

	genMostReceivedDislikesMember, err := ls.db.query.MostDislikesReceived(context.TODO(), date.String())
	if err != nil {
		return nil, err
	}
	mostReceivedDislikesMember, err := genMemberToMember(&genMostReceivedDislikesMember)
	if err != nil {
		return nil, err
	}

	return &stats.Leaderboard{
		Date:                       date,
		MostReceivedLikesMember:    *mostReceivedLikesMember,
		MostReceivedDislikesMember: *mostReceivedDislikesMember,
	}, nil
}
