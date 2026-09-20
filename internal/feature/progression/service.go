package progression

import "context"

// Granter is the narrow slice of this feature that grading depends on.
// It lives here so the name states the domain intent, but consumers are
// free to redeclare it at their own point of use — see ielts_test.
type Granter interface {
	GrantIfFirstAttempt(ctx context.Context, userID, testID, submissionID uint64, amount int) (granted bool, level int, xp int, err error)
}

type Service interface {
	Get(ctx context.Context, userID uint64) (*Progression, error)
	// SetEquippedFrame equips an avatar frame, refusing any the user has
	// not unlocked yet.
	SetEquippedFrame(ctx context.Context, userID uint64, frameLevel int) (*Progression, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Get(ctx context.Context, userID uint64) (*Progression, error) {
	return s.repo.Get(ctx, userID)
}

func (s *service) SetEquippedFrame(ctx context.Context, userID uint64, frameLevel int) (*Progression, error) {
	p, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	// The gate reads the level derived from XP, not the cached column —
	// so a stale users.level can never unlock a frame the user has not
	// actually earned, nor lock one they have.
	if !p.CanEquip(frameLevel) {
		return nil, ErrFrameLocked
	}
	if err := s.repo.SetEquippedFrame(ctx, userID, frameLevel); err != nil {
		return nil, err
	}
	p.EquippedFrameLevel = &frameLevel
	return p, nil
}

var _ Granter = (Repository)(nil)
