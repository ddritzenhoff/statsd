package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ddritzenhoff/statsd"
	"github.com/ddritzenhoff/statsd/sqlite/gen"
)

// Ensure service implements interface.
var _ statsd.MemberService = (*MemberService)(nil)

// MemberService represents a service for managing Members.
type MemberService struct {
	db *DB
}

// NewMemberService returns a new instance of MemberService.
func NewMemberService(db *DB) *MemberService {
	return &MemberService{
		db: db,
	}
}

// FindMemberByID retrieves a Member by ID.
// Returns ErrNotFound if the ID does not exist.
func (ms *MemberService) FindMemberByID(id int) (*statsd.Member, error) {
	tx, err := ms.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// fetch member
	genMember, err := ms.db.query.FindMemberByID(context.TODO(), int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, statsd.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return genMemberToMember(&genMember)
}

// FindMember retrives a Member by his Slack User ID, the Month, and the Year.
// Returns ErrNotFound if not matches found.
func (ms *MemberService) FindMember(SlackUID string, date statsd.MonthYear) (*statsd.Member, error) {
	tx, err := ms.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	genMember, err := ms.db.query.FindMember(context.TODO(), gen.FindMemberParams{
		SlackUid:  SlackUID,
		MonthYear: date.String(),
	})

	if errors.Is(err, sql.ErrNoRows) {
		return nil, statsd.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return genMemberToMember(&genMember)
}

// CreateMember creates a new Member.
func (ms *MemberService) CreateMember(m *statsd.Member) error {
	tx, err := ms.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if m == nil {
		return fmt.Errorf("CreateMember: m reference is nil")
	}

	if err := m.Validate(); err != nil {
		return err
	}

	m.CreatedAt = tx.now
	m.UpdatedAt = m.CreatedAt

	genMem, err := ms.db.query.CreateMember(context.TODO(), gen.CreateMemberParams{
		SlackUid:  m.SlackUID,
		MonthYear: m.Date.String(),
		CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: m.UpdatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("CreateMember: %w", err)
	}

	m.ID = int(genMem.ID)
	m.ReceivedDislikes = int(genMem.ReceivedDislikes)
	m.ReceivedLikes = int(genMem.ReceivedLikes)

	return tx.Commit()
}

// UpdateMember updates a Member.
// Returns ErrNotFound if the member does not exist.
func (ms *MemberService) UpdateMember(id int, upd statsd.MemberUpdate) (*statsd.Member, error) {
	tx, err := ms.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("UpdateMember db.Begin: %w", err)
	}
	defer tx.Rollback()

	m, err := ms.FindMemberByID(id)
	if err != nil {
		return nil, err
	}

	if v := upd.ReceivedLikes; v != nil {
		m.ReceivedLikes = *v
	}
	if v := upd.ReceivedDislikes; v != nil {
		m.ReceivedDislikes = *v
	}

	genMem, err := ms.db.query.UpdateMember(context.TODO(), gen.UpdateMemberParams{
		ReceivedLikes:    int64(m.ReceivedLikes),
		ReceivedDislikes: int64(m.ReceivedDislikes),
		UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
		ID:               int64(id),
	})
	if err != nil {
		return nil, fmt.Errorf("UpdateMember: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return genMemberToMember(&genMem)
}

// DeleteMember permanently deletes a Member.
func (ms *MemberService) DeleteMember(id int) error {
	tx, err := ms.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = ms.db.query.DeleteMember(context.TODO(), int64(id))
	if err != nil {
		return fmt.Errorf("DeleteMember: %w", err)
	}
	return nil
}

// genMemberToMember converts the sqlite member type to the stats member type.
func genMemberToMember(mem *gen.Member) (*statsd.Member, error) {
	date, err := statsd.NewMonthYearString(mem.MonthYear)
	if err != nil {
		return nil, err
	}
	createdAt, err := time.Parse(time.RFC3339, mem.CreatedAt)
	if err != nil {
		return nil, err
	}
	updatedAt, err := time.Parse(time.RFC3339, mem.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &statsd.Member{ID: int(mem.ID), Date: date, SlackUID: mem.SlackUid, ReceivedLikes: int(mem.ReceivedLikes), ReceivedDislikes: int(mem.ReceivedDislikes), CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}
