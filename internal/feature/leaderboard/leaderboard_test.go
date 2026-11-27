package leaderboard

import (
	"context"
	"testing"
)

type fakeRepo struct{ stats []Stats }

func (f *fakeRepo) All(ctx context.Context) ([]Stats, error) {
	return append([]Stats{}, f.stats...), nil
}

func newBoard(stats ...Stats) Service { return NewService(&fakeRepo{stats: stats}) }

func ids(entries []Entry) []uint64 {
	out := make([]uint64, len(entries))
	for i, e := range entries {
		out[i] = e.UserID
	}
	return out
}

func equalIDs(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var sample = []Stats{
	{UserID: 1, Name: "An", XP: 300, Reading: 1},
	{UserID: 2, Name: "Bình", XP: 900, Reading: 1, Writing: 1},
	{UserID: 3, Name: "Chi", XP: 120, Reading: 3, Listening: 2, Speaking: 1},
	{UserID: 4, Name: "Dũng", XP: 300, Listening: 4},
}

func TestRanksByXPWithAttemptsAsTieBreak(t *testing.T) {
	b, err := newBoard(sample...).Board(context.Background(), SortXP, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ids(b.Entries), []uint64{2, 4, 1, 3}; !equalIDs(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
	// 4 and 1 have the same XP, so they share rank 2 and the next is 4th.
	ranks := [4]int{b.Entries[0].Rank, b.Entries[1].Rank, b.Entries[2].Rank, b.Entries[3].Rank}
	if ranks != [4]int{1, 2, 2, 4} {
		t.Errorf("ranks = %v, want [1 2 2 4]", ranks)
	}
}

func TestRanksByAttempts(t *testing.T) {
	b, err := newBoard(sample...).Board(context.Background(), SortAttempts, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ids(b.Entries), []uint64{3, 4, 2, 1}; !equalIDs(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
	first := b.Entries[0]
	if first.Attempts != 6 || first.Skills.Reading != 3 || first.Skills.Listening != 2 || first.Skills.Speaking != 1 {
		t.Errorf("first row = %+v, want 6 attempts (3 reading, 2 listening, 1 speaking)", first)
	}
}

func TestCallerBelowTheCutComesBackAsMe(t *testing.T) {
	b, err := newBoard(sample...).Board(context.Background(), SortXP, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Entries) != 2 || b.Total != 4 {
		t.Fatalf("entries = %d, total = %d; want 2 and 4", len(b.Entries), b.Total)
	}
	if b.Me == nil || b.Me.UserID != 3 || b.Me.Rank != 4 || !b.Me.IsMe {
		t.Errorf("me = %+v, want user 3 at rank 4", b.Me)
	}
}

func TestCallerInTheTopIsMarkedNotRepeated(t *testing.T) {
	b, err := newBoard(sample...).Board(context.Background(), SortXP, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !b.Entries[0].IsMe || b.Me != nil {
		t.Errorf("caller in the top: IsMe = %v, Me = %+v; want true and nil", b.Entries[0].IsMe, b.Me)
	}
}

func TestUnknownSortFallsBackToXP(t *testing.T) {
	b, err := newBoard(sample...).Board(context.Background(), Sort("banana"), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if b.Sort != SortXP || b.Entries[0].UserID != 2 {
		t.Errorf("sort = %q, first = %d; want xp and user 2", b.Sort, b.Entries[0].UserID)
	}
}
