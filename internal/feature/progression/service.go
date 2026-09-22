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

var _ Granter = (Repository)(nil)
