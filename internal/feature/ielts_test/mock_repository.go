package ielts_test

import (
	"context"
	"sort"
	"time"
)

type MockTestRepository struct {
	tests       map[int]*Test
	submissions map[int]*Submission
	scores      map[int]*Score

	nextTestID       uint64
	nextSubmissionID uint64
	nextScoreID      uint64
}

func NewMockTestRepository() *MockTestRepository {
	return &MockTestRepository{tests: make(map[int]*Test), submissions: make(map[int]*Submission), scores: make(map[int]*Score)}
}

// CreateTest mimics an auto-incrementing primary key: a caller that leaves
// ID unset (the common case) gets the next sequential ID assigned; a caller
// that pre-seeds a specific ID (for test fixtures) is left alone.
func (r *MockTestRepository) CreateTest(ctx context.Context, t *Test) (*Test, error) {
	if t.ID == 0 {
		r.nextTestID++
		t.ID = r.nextTestID
	}
	r.tests[int(t.ID)] = t
	return t, nil
}
func (r *MockTestRepository) GetTestByID(ctx context.Context, id uint64) (*Test, error) {
	test, ok := r.tests[int(id)]
	if !ok {
		return nil, ErrTestNotFound
	}

	return test, nil
}

func (r *MockTestRepository) GetListTest(ctx context.Context, skill string, limit, offset int) ([]Test, int, error) {
	var filtered []Test

	for _, t := range r.tests {
		if skill == "" || t.Skill == skill {
			filtered = append(filtered, *t)
		}
	}

	total := len(filtered)

	if offset >= total {
		return []Test{}, total, nil
	}

	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}

	return filtered[offset:end], total, nil
}

func (r *MockTestRepository) CreateSubmission(ctx context.Context, s *Submission) (*Submission, error) {
	if s.ID == 0 {
		r.nextSubmissionID++
		s.ID = r.nextSubmissionID
	}
	r.submissions[int(s.ID)] = s
	return s, nil
}

func (r *MockTestRepository) GetSubmissionByID(ctx context.Context, id uint64) (*Submission, error) {
	submission, ok := r.submissions[int(id)]
	if !ok {
		return nil, ErrSubmissionNotFound
	}
	return submission, nil
}

func (r *MockTestRepository) GetListSubmission(ctx context.Context, userID uint64, limit, offset int) ([]SubmissionSummary, int, error) {
	var summaries []SubmissionSummary
	for _, s := range r.submissions {
		if s.UserID != userID {
			continue
		}
		test, err := r.GetTestByID(ctx, s.TestID)
		if err != nil {
			return nil, 0, err
		}
		// Mirrors the real repository's LEFT JOIN scores: a submission with
		// no score yet (still pending/failed) is included with a nil band
		// rather than being dropped from the list.
		var overallBand *float64
		var gradedAt *time.Time
		if score, err := r.GetScoreBySubmissionID(ctx, s.ID); err == nil {
			overallBand = score.OverallBand
			gradedAt = score.GradedAt
		}
		summaries = append(summaries, SubmissionSummary{
			*s,
			test.Skill,
			test.TaskType,
			test.XPGain,
			overallBand,
			gradedAt,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].SubmittedAt.After(summaries[j].SubmittedAt)
	})

	total := len(summaries)

	if offset >= total {
		return []SubmissionSummary{}, total, nil
	}

	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}

	return summaries[offset:end], total, nil
}

func (r *MockTestRepository) UpdateSubmissionStatus(ctx context.Context, id uint64, status string) error {
	submission, ok := r.submissions[int(id)]
	if !ok {
		return ErrSubmissionNotFound
	}

	submission.Status = status
	return nil
}

func (r *MockTestRepository) CreateScore(ctx context.Context, sc *Score) (*Score, error) {
	if sc.ID == 0 {
		r.nextScoreID++
		sc.ID = r.nextScoreID
	}
	r.scores[int(sc.SubmissionID)] = sc
	return sc, nil
}

func (r *MockTestRepository) GetScoreBySubmissionID(ctx context.Context, submissionID uint64) (*Score, error) {
	score, ok := r.scores[int(submissionID)]
	if !ok {
		return nil, ErrScoreNotFound
	}

	return score, nil
}
