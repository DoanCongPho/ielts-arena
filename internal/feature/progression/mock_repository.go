package progression

import "context"

// MockRepository is an in-memory Repository for tests, including tests in
// other packages that need something to satisfy the interface.
type MockRepository struct {
	Progressions map[uint64]*Progression
	// Grants records which (userID, testID) pairs have already been
	// awarded, mirroring the submission_xp_grants primary key that makes
	// the real grant idempotent.
	Grants map[[2]uint64]bool
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		Progressions: make(map[uint64]*Progression),
		Grants:       make(map[[2]uint64]bool),
	}
}

// Seed creates a user's progression from a lifetime XP total.
func (m *MockRepository) Seed(userID uint64, xp int) *Progression {
	p := &Progression{UserID: userID, XP: xp}
	p.Level, _, _ = levelForXP(xp)
	m.Progressions[userID] = p
	return p
}

func (m *MockRepository) Get(ctx context.Context, userID uint64) (*Progression, error) {
	p, ok := m.Progressions[userID]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MockRepository) GrantIfFirstAttempt(ctx context.Context, userID, testID, submissionID uint64, amount int) (bool, int, int, error) {
	p, ok := m.Progressions[userID]
	if !ok {
		return false, 0, 0, ErrNotFound
	}
	key := [2]uint64{userID, testID}
	if m.Grants[key] {
		return false, p.Level, p.XP, nil
	}
	m.Grants[key] = true
	p.XP += amount
	p.Level, _, _ = levelForXP(p.XP)
	return true, p.Level, p.XP, nil
}

func (m *MockRepository) SetEquippedFrame(ctx context.Context, userID uint64, frameLevel int) error {
	p, ok := m.Progressions[userID]
	if !ok {
		return ErrNotFound
	}
	p.EquippedFrameLevel = &frameLevel
	return nil
}

var _ Repository = (*MockRepository)(nil)
