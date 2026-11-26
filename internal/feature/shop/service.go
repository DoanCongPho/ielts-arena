// Package shop lets a learner spend XP on cosmetic items — avatars and
// avatar frames — and wear them. Spending never lowers the level: the
// level comes from lifetime XP (feature/progression), and the shop keeps
// its own tally of what has been spent.
package shop

import (
	"context"
	"time"
)

// ItemView is a catalog item as one user sees it.
type ItemView struct {
	Item
	Owned    bool `json:"owned"`
	Equipped bool `json:"equipped"`
	OnSale   bool `json:"on_sale"`
}

type ShopResponse struct {
	Balance        int        `json:"xp_balance"`
	EquippedAvatar *string    `json:"equipped_avatar"`
	EquippedFrame  *string    `json:"equipped_frame"`
	Items          []ItemView `json:"items"`
}

type Service interface {
	Shop(ctx context.Context, userID uint64) (*ShopResponse, error)
	Purchase(ctx context.Context, userID uint64, itemID string) (*ShopResponse, error)
	// Equip wears itemID in the `kind` slot; an empty itemID takes off
	// whatever is there.
	Equip(ctx context.Context, userID uint64, kind Kind, itemID string) (*ShopResponse, error)
	Wallet(ctx context.Context, userID uint64) (*Wallet, error)
}

type Option func(*service)

// WithClock replaces time.Now, for tests.
func WithClock(now func() time.Time) Option {
	return func(s *service) { s.now = now }
}

// WithOpenSeasons puts the listed seasons on sale whatever the date — for
// trying the shop out locally (SHOP_FORCE_SEASON).
func WithOpenSeasons(seasons ...Season) Option {
	return func(s *service) {
		for _, se := range seasons {
			s.openSeasons[se] = true
		}
	}
}

type service struct {
	repo        Repository
	now         func() time.Time
	openSeasons map[Season]bool
}

func NewService(repo Repository, opts ...Option) Service {
	s := &service{repo: repo, now: time.Now, openSeasons: map[Season]bool{}}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *service) onSale(it Item) bool {
	return s.openSeasons[it.Season] || inSeason(it.Season, s.now())
}

func (s *service) Wallet(ctx context.Context, userID uint64) (*Wallet, error) {
	return s.repo.Wallet(ctx, userID)
}

func (s *service) Shop(ctx context.Context, userID uint64) (*ShopResponse, error) {
	w, err := s.repo.Wallet(ctx, userID)
	if err != nil {
		return nil, err
	}
	owned, err := s.repo.Owned(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.view(w, owned), nil
}

func (s *service) view(w *Wallet, owned []string) *ShopResponse {
	has := make(map[string]bool, len(owned))
	for _, id := range owned {
		has[id] = true
	}
	items := make([]ItemView, 0, len(catalog))
	for _, it := range catalog {
		items = append(items, ItemView{
			Item:     it,
			Owned:    has[it.ID],
			Equipped: equals(w.EquippedAvatar, it.ID) || equals(w.EquippedFrame, it.ID),
			OnSale:   s.onSale(it),
		})
	}
	return &ShopResponse{
		Balance:        w.Balance(),
		EquippedAvatar: w.EquippedAvatar,
		EquippedFrame:  w.EquippedFrame,
		Items:          items,
	}
}

func (s *service) Purchase(ctx context.Context, userID uint64, itemID string) (*ShopResponse, error) {
	it, ok := findItem(itemID)
	if !ok {
		return nil, ErrUnknownItem
	}
	if !s.onSale(it) {
		return nil, ErrOutOfSeason
	}
	if _, err := s.repo.Purchase(ctx, userID, it, s.now()); err != nil {
		return nil, err
	}
	return s.Shop(ctx, userID)
}

func (s *service) Equip(ctx context.Context, userID uint64, kind Kind, itemID string) (*ShopResponse, error) {
	if kind != KindAvatar && kind != KindFrame {
		return nil, ErrWrongSlot
	}
	var target *string
	if itemID != "" {
		it, ok := findItem(itemID)
		if !ok {
			return nil, ErrUnknownItem
		}
		if it.Kind != kind {
			return nil, ErrWrongSlot
		}
		owned, err := s.repo.Owned(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !contains(owned, itemID) {
			return nil, ErrNotOwned
		}
		target = &itemID
	}
	if err := s.repo.Equip(ctx, userID, kind, target); err != nil {
		return nil, err
	}
	return s.Shop(ctx, userID)
}

func equals(p *string, v string) bool { return p != nil && *p == v }

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
