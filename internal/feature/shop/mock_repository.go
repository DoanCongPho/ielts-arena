package shop

import (
	"context"
	"time"
)

// MockRepository is an in-memory Repository for tests.
type MockRepository struct {
	Wallets map[uint64]*Wallet
	Items   map[uint64][]string
}

func NewMockRepository() *MockRepository {
	return &MockRepository{Wallets: map[uint64]*Wallet{}, Items: map[uint64][]string{}}
}

func (m *MockRepository) Seed(userID uint64, xp int) *Wallet {
	w := &Wallet{XP: xp}
	m.Wallets[userID] = w
	return w
}

func (m *MockRepository) Wallet(ctx context.Context, userID uint64) (*Wallet, error) {
	w, ok := m.Wallets[userID]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *w
	return &cp, nil
}

func (m *MockRepository) Owned(ctx context.Context, userID uint64) ([]string, error) {
	return append([]string{}, m.Items[userID]...), nil
}

func (m *MockRepository) Purchase(ctx context.Context, userID uint64, item Item, now time.Time) (*Wallet, error) {
	w, ok := m.Wallets[userID]
	if !ok {
		return nil, ErrNotFound
	}
	if contains(m.Items[userID], item.ID) {
		return nil, ErrAlreadyOwned
	}
	if w.Balance() < item.Price {
		return nil, ErrInsufficientXP
	}
	m.Items[userID] = append(m.Items[userID], item.ID)
	w.Spent += item.Price
	cp := *w
	return &cp, nil
}

func (m *MockRepository) Equip(ctx context.Context, userID uint64, kind Kind, itemID *string) error {
	w, ok := m.Wallets[userID]
	if !ok {
		return ErrNotFound
	}
	switch kind {
	case KindAvatar:
		w.EquippedAvatar = itemID
	case KindFrame:
		w.EquippedFrame = itemID
	default:
		return ErrWrongSlot
	}
	return nil
}

var _ Repository = (*MockRepository)(nil)
