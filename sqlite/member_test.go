package sqlite_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ddritzenhoff/statsd"
	"github.com/ddritzenhoff/statsd/sqlite"
)

func TestMemberService_CreateMember(t *testing.T) {
	// Ensure user can be created.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		ms := sqlite.NewMemberService(db)

		monthYear, err := statsd.NewMonthYearString("2006-05")
		if err != nil {
			t.Fatal(err)
		}
		m := &statsd.Member{
			Date:     monthYear,
			SlackUID: "U1ZN1SE2N",
		}

		// Create new user & verify ID and timestamps are set.
		if err := ms.CreateMember(m); err != nil {
			t.Fatal(err)
		} else if got, want := m.ID, 1; got != want {
			t.Fatalf("ID=%v, want %v", got, want)
		} else if got, want := m.ReceivedLikes, 0; got != want {
			t.Fatalf("received likes=%v, want=%v", m.ReceivedLikes, 0)
		} else if got, want := m.ReceivedDislikes, 0; got != want {
			t.Fatalf("received dislikes=%v, want=%v", m.ReceivedDislikes, 0)
		} else if m.CreatedAt.IsZero() {
			t.Fatal("expected created at")
		} else if m.UpdatedAt.IsZero() {
			t.Fatal("expected updated at")
		}

		// Create second user with email.
		monthYear, err = statsd.NewMonthYearString("2006-05")
		if err != nil {
			t.Fatal(err)
		}
		m2 := &statsd.Member{
			Date:     monthYear,
			SlackUID: "U2ZN1SE2N",
		}
		if err := ms.CreateMember(m2); err != nil {
			t.Fatal(err)
		} else if got, want := m2.ID, 2; got != want {
			t.Fatalf("ID=%v, want %v", got, want)
		}

		// Fetch user from database & compare.
		if other, err := ms.FindMemberByID(1); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(m, other) {
			t.Fatalf("mismatch: %#v != %#v", m, other)
		}
	})
	// Ensure an error is returned if user name is not set.
	t.Run("ErrNameRequired", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		ms := sqlite.NewMemberService(db)

		if err := ms.CreateMember(&statsd.Member{}); err == nil {
			t.Fatal("expected error")
		} else if !errors.Is(err, statsd.ErrInvalid) {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

func TestMemberService_UpdateMember(t *testing.T) {
	// Ensure user name & email can be updated by current user.
	t.Run("OK", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)

		ms := sqlite.NewMemberService(db)
		m1 := MustCreateMember(t, db, &statsd.Member{
			Date:     statsd.MonthYear("2006-05"),
			SlackUID: "U2ZN1SE2N",
		})

		newReceivedLikes := 5
		newReceivedDislikes := 23
		m2, err := ms.UpdateMember(m1.ID, statsd.MemberUpdate{
			ReceivedLikes:    &newReceivedLikes,
			ReceivedDislikes: &newReceivedDislikes,
		})

		if err != nil {
			t.Fatal(err)
		} else if got, want := m2.ReceivedLikes, newReceivedLikes; got != want {
			t.Fatalf("ReceivedLikes=%v, want %v", got, want)
		} else if got, want := m2.ReceivedDislikes, newReceivedDislikes; got != want {
			t.Fatalf("ReceivedDislikes=%v, want %v", got, want)
		}

		// Fetch user from database & compare.
		if other, err := ms.FindMemberByID(1); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(m2, other) {
			t.Fatalf("mismatch: %#v != %#v", m2, other)
		}
	})
}

func TestMemberService_FindMember(t *testing.T) {
	// Ensure an error is returned if fetching a non-existent user.
	t.Run("ErrNotFound FindMemberByID", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		ms := sqlite.NewMemberService(db)
		if _, err := ms.FindMemberByID(1); !errors.Is(err, statsd.ErrNotFound) {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
	t.Run("ErrNotFound FindMember", func(t *testing.T) {
		db := MustOpenDB(t)
		defer MustCloseDB(t, db)
		ms := sqlite.NewMemberService(db)
		if _, err := ms.FindMember("abc123", statsd.MonthYear("hey")); !errors.Is(err, statsd.ErrNotFound) {
			t.Fatalf("unexpected error: %#v", err)
		}
	})
}

// MustCreateMember creates a member in the database. Fatal on error.
func MustCreateMember(tb testing.TB, db *sqlite.DB, m *statsd.Member) *statsd.Member {
	tb.Helper()
	if err := sqlite.NewMemberService(db).CreateMember(m); err != nil {
		tb.Fatal(err)
	}
	return m
}
