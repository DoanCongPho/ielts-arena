package leaderboard

import (
	"context"
	"sort"

	"github/DoanCongPho/game-arena/internal/platform/leveling"
)

const (
	DefaultLimit = 50
	MaxLimit     = 100
)

type Service interface {
	// Board ranks everyone by `by` and returns the top `limit`, marking
	// the caller's row; if the caller is ranked below the cut, their row
	// comes back in Board.Me.
	Board(ctx context.Context, by Sort, userID uint64, limit int) (*Board, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Board(ctx context.Context, by Sort, userID uint64, limit int) (*Board, error) {
	if by != SortAttempts {
		by = SortXP
	}
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	all, err := s.repo.All(ctx)
	if err != nil {
		return nil, err
	}

	// Primary key first, the other as tie-break, then oldest account, so
	// the order is stable between requests.
	primary := func(st Stats) int { return st.XP }
	secondary := func(st Stats) int { return st.Attempts() }
	if by == SortAttempts {
		primary, secondary = secondary, primary
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if primary(a) != primary(b) {
			return primary(a) > primary(b)
		}
		if secondary(a) != secondary(b) {
			return secondary(a) > secondary(b)
		}
		return a.UserID < b.UserID
	})

	board := &Board{Sort: by, Entries: []Entry{}, Total: len(all)}
	rank := 0
	for i, st := range all {
		// Standard competition ranking: equal scores share a rank (1, 2, 2, 4).
		if i == 0 || primary(st) != primary(all[i-1]) {
			rank = i + 1
		}
		e := toEntry(st, rank, st.UserID == userID)
		if i < limit {
			board.Entries = append(board.Entries, e)
		} else if e.IsMe {
			board.Me = &e
		}
	}
	return board, nil
}

func toEntry(st Stats, rank int, isMe bool) Entry {
	level, _, _ := leveling.LevelForXP(st.XP)
	return Entry{
		Rank:           rank,
		UserID:         st.UserID,
		Name:           st.Name,
		ImageURL:       st.ImageURL,
		EquippedAvatar: st.EquippedAvatar,
		EquippedFrame:  st.EquippedFrame,
		Level:          level,
		XP:             st.XP,
		Attempts:       st.Attempts(),
		Skills: Skills{
			Reading:   st.Reading,
			Listening: st.Listening,
			Writing:   st.Writing,
			Speaking:  st.Speaking,
		},
		IsMe: isMe,
	}
}
