package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ddritzenhoff/stats"
	"github.com/ddritzenhoff/stats/sqlite/gen"
)

// Ensure service implements interface.
var _ stats.MemberService = (*MemberService)(nil)

// MemberService represents a service for managing Members.
type MemberService struct {
	query *gen.Queries
	db    *sql.DB
}

// NewMemberService returns a new instance of MemberService.
func NewMemberService(query *gen.Queries, db *sql.DB) *MemberService {
	return &MemberService{
		query: query,
		db:    db,
	}
}

// FindMemberByID retrieves a Member by ID.
// Returns ErrNotFound if the ID does not exist.
func (ms *MemberService) FindMemberByID(id int64) (*stats.Member, error) {
	genMember, err := ms.query.FindMemberByID(context.TODO(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, stats.ErrNotFound
		}
		return nil, err
	}

	return &stats.Member{
		ID:                genMember.ID,
		SlackUID:          genMember.SlackUid,
		ReceivedLikes:     genMember.ReceivedLikes,
		ReceivedDislikes:  genMember.ReceivedDislikes,
		ReceivedReactions: genMember.ReceivedReactions,
		GivenLikes:        genMember.GivenLikes,
		GivenDislikes:     genMember.GivenDislikes,
		GivenReactions:    genMember.GivenReactions,
		Date: stats.Date{
			Month: time.Month(genMember.Month),
			Year:  genMember.Year,
		},
	}, nil
}

// FindMember retrives a Member by his Slack User ID, the Month, and the Year.
// Returns ErrNotFound if not matches found.
func (ms *MemberService) FindMember(SlackUID string, Month time.Month, Year int64) (*stats.Member, error) {
	genMember, err := ms.query.FindMember(context.TODO(), gen.FindMemberParams{
		SlackUid: SlackUID,
		Month:    int64(Month),
		Year:     Year,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, stats.ErrNotFound
		}
		return nil, err
	}

	return &stats.Member{
		ID:                genMember.ID,
		SlackUID:          genMember.SlackUid,
		ReceivedLikes:     genMember.ReceivedLikes,
		ReceivedDislikes:  genMember.ReceivedDislikes,
		ReceivedReactions: genMember.ReceivedReactions,
		GivenLikes:        genMember.GivenLikes,
		GivenDislikes:     genMember.GivenDislikes,
		GivenReactions:    genMember.GivenReactions,
		Date: stats.Date{
			Month: time.Month(genMember.Month),
			Year:  genMember.Year,
		},
	}, nil
}

// Summary returns a summary of the members.
func (ms *MemberService) Summary() (*stats.MemberSummary, error) {
	year, month, _ := time.Now().Date()
	date := stats.Date{
		Month: month,
		Year:  int64(year),
	}
	summary := &stats.MemberSummary{}

	// Most likes received.
	members, err := ms.mostLikesReceived(date)
	if err != nil {
		return nil, fmt.Errorf("mostLikesReceived: %w", err)
	}
	summary.MostLikesReceived = members

	// Most dislikes received.
	members, err = ms.mostDislikesReceived(date)
	if err != nil {
		return nil, fmt.Errorf("mostDislikesReceived: %w", err)
	}
	summary.MostDislikesReceived = members

	// Most reactions received.
	members, err = ms.mostReactionsReceived(date)
	if err != nil {
		return nil, fmt.Errorf("mostReactionsReceived: %w", err)
	}
	summary.MostReactionsReceived = members

	// Most likes given.
	members, err = ms.mostLikesGiven(date)
	if err != nil {
		return nil, fmt.Errorf("mostLikesGiven: %w", err)
	}
	summary.MostLikesGiven = members

	// Most dislikes given.
	members, err = ms.mostDislikesGiven(date)
	if err != nil {
		return nil, fmt.Errorf("mostDislikesGiven: %w", err)
	}
	summary.MostDislikesGiven = members

	// Most reactions given.
	members, err = ms.mostReactionsGiven(date)
	if err != nil {
		return nil, fmt.Errorf("mostReactionsGiven: %w", err)
	}
	summary.MostReactionsGiven = members

	return summary, nil
}

// mostDislikesReceived returns the member with the most dislikes received within a given month and year.
func (ms *MemberService) mostDislikesReceived(date stats.Date) ([]stats.Member, error) {
	genMembers, err := ms.query.MostDislikesReceived(context.TODO(), gen.MostDislikesReceivedParams{
		Month:   int64(date.Month),
		Year:    date.Year,
		Month_2: int64(date.Month),
		Year_2:  date.Year,
	})
	if err != nil {
		return nil, fmt.Errorf("mostDislikesReceived: %w", err)
	}
	var members []stats.Member
	for _, member := range genMembers {
		members = append(members, stats.Member{
			ID:               member.ID,
			SlackUID:         member.SlackUid,
			ReceivedDislikes: member.ReceivedDislikes,
			Date: stats.Date{
				Month: time.Month(member.Month),
				Year:  member.Year,
			},
		})
	}
	return members, nil
}

// mostDislikesGiven returns the member with the most dislikes given within a given month and year.
func (ms *MemberService) mostDislikesGiven(date stats.Date) ([]stats.Member, error) {
	genMembers, err := ms.query.MostDislikesGiven(context.TODO(), gen.MostDislikesGivenParams{
		Month:   int64(date.Month),
		Year:    date.Year,
		Month_2: int64(date.Month),
		Year_2:  date.Year,
	})
	if err != nil {
		return nil, fmt.Errorf("mostDislikesGiven: %w", err)
	}
	var members []stats.Member
	for _, member := range genMembers {
		members = append(members, stats.Member{
			ID:            member.ID,
			SlackUID:      member.SlackUid,
			GivenDislikes: member.GivenDislikes,
			Date: stats.Date{
				Month: time.Month(member.Month),
				Year:  member.Year,
			},
		})
	}
	return members, nil
}

// mostReactionsGiven returns the member with the most reactions given within a given month and year.
func (ms *MemberService) mostReactionsGiven(date stats.Date) ([]stats.Member, error) {
	genMembers, err := ms.query.MostReactionsGiven(context.TODO(), gen.MostReactionsGivenParams{
		Month:   int64(date.Month),
		Year:    date.Year,
		Month_2: int64(date.Month),
		Year_2:  date.Year,
	})
	if err != nil {
		return nil, fmt.Errorf("mostReactionsGiven: %w", err)
	}
	var members []stats.Member
	for _, member := range genMembers {
		members = append(members, stats.Member{
			ID:             member.ID,
			SlackUID:       member.SlackUid,
			GivenReactions: member.GivenReactions,
			Date: stats.Date{
				Month: time.Month(member.Month),
				Year:  member.Year,
			},
		})
	}
	return members, nil
}

// mostReactionsReceived returns the member with the most reactions received within a given month and year.
func (ms *MemberService) mostReactionsReceived(date stats.Date) ([]stats.Member, error) {
	genMembers, err := ms.query.MostReactionsReceived(context.TODO(), gen.MostReactionsReceivedParams{
		Month:   int64(date.Month),
		Year:    date.Year,
		Month_2: int64(date.Month),
		Year_2:  date.Year,
	})
	if err != nil {
		return nil, fmt.Errorf("mostReactionsReceived: %w", err)
	}
	var members []stats.Member
	for _, member := range genMembers {
		members = append(members, stats.Member{
			ID:                member.ID,
			SlackUID:          member.SlackUid,
			ReceivedReactions: member.ReceivedReactions,
			Date: stats.Date{
				Month: time.Month(member.Month),
				Year:  member.Year,
			},
		})
	}
	return members, nil
}

// mostLikesGiven returns the member with the most likes given within a given month and year.
func (ms *MemberService) mostLikesGiven(date stats.Date) ([]stats.Member, error) {
	genMembers, err := ms.query.MostLikesGiven(context.TODO(), gen.MostLikesGivenParams{
		Month:   int64(date.Month),
		Year:    date.Year,
		Month_2: int64(date.Month),
		Year_2:  date.Year,
	})
	if err != nil {
		return nil, fmt.Errorf("mostLikesGiven: %w", err)
	}
	var members []stats.Member
	for _, member := range genMembers {
		members = append(members, stats.Member{
			ID:         member.ID,
			SlackUID:   member.SlackUid,
			GivenLikes: member.GivenLikes,
			Date: stats.Date{
				Month: time.Month(member.Month),
				Year:  member.Year,
			},
		})
	}
	return members, nil
}

// mostLikesReceived returns the member with the most likes received within a given month and year.
func (ms *MemberService) mostLikesReceived(date stats.Date) ([]stats.Member, error) {
	genMembers, err := ms.query.MostLikesReceived(context.TODO(), gen.MostLikesReceivedParams{
		Month:   int64(date.Month),
		Year:    date.Year,
		Month_2: int64(date.Month),
		Year_2:  date.Year,
	})
	if err != nil {
		return nil, fmt.Errorf("mostLikesReceived: %w", err)
	}
	var members []stats.Member
	for _, member := range genMembers {
		members = append(members, stats.Member{
			ID:            member.ID,
			SlackUID:      member.SlackUid,
			ReceivedLikes: member.ReceivedLikes,
			Date: stats.Date{
				Month: time.Month(member.Month),
				Year:  member.Year,
			},
		})
	}
	return members, nil
}

// CreateMember creates a new Member.
func (ms *MemberService) CreateMember(m *stats.Member) error {
	if m == nil {
		return fmt.Errorf("CreateMember: m reference is nil")
	}
	_, err := ms.query.CreateMember(context.TODO(), gen.CreateMemberParams{
		SlackUid: m.SlackUID,
		Month:    int64(m.Date.Month),
		Year:     m.Date.Year,
	})
	if err != nil {
		return fmt.Errorf("CreateMember: %w", err)
	}
	return nil
}

// UpdateMember updates a Member.
func (ms *MemberService) UpdateMember(id int64, upd stats.MemberUpdate) error {
	tx, err := ms.db.Begin()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("UpdateMember db.Begin: %w", err)
	}

	genMember, err := ms.query.FindMemberByID(context.TODO(), id)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("UpdateMember FindMemberByID: %w", err)
	}

	if upd.ReceivedLikes != nil {
		genMember.ReceivedLikes = *upd.ReceivedLikes
	}
	if upd.ReceivedDislikes != nil {
		genMember.ReceivedDislikes = *upd.ReceivedDislikes
	}
	if upd.ReceivedReactions != nil {
		genMember.ReceivedReactions = *upd.ReceivedReactions
	}
	if upd.GivenLikes != nil {
		genMember.GivenLikes = *upd.GivenLikes
	}
	if upd.GivenDislikes != nil {
		genMember.GivenDislikes = *upd.GivenDislikes
	}
	if upd.GivenReactions != nil {
		genMember.GivenReactions = *upd.GivenReactions
	}

	err = ms.query.UpdateMember(context.TODO(), gen.UpdateMemberParams{
		ReceivedLikes:     genMember.ReceivedLikes,
		ReceivedDislikes:  genMember.ReceivedDislikes,
		ReceivedReactions: genMember.ReceivedReactions,
		GivenLikes:        genMember.GivenLikes,
		GivenDislikes:     genMember.GivenDislikes,
		GivenReactions:    genMember.GivenReactions,
		ID:                id,
	})
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("UpdateMember: %w", err)
	}

	tx.Commit()
	return nil
}

// DeleteMember permanently deletes a Member
func (ms *MemberService) DeleteMember(id int64) error {
	err := ms.query.DeleteMember(context.TODO(), id)
	if err != nil {
		return fmt.Errorf("DeleteMember: %w", err)
	}
	return nil
}
