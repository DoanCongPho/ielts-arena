// Package leaderboard ranks learners by lifetime XP or by how many tests
// they've finished, with a per-skill breakdown. It owns no table: it reads
// users (XP, name, worn shop items) and graded submissions.
package leaderboard

// Stats is one learner's standing, before ranking.
type Stats struct {
	UserID         uint64
	Name           string
	ImageURL       string
	EquippedAvatar *string
	EquippedFrame  *string
	XP             int
	Reading        int
	Listening      int
	Writing        int
	Speaking       int
}

// Attempts is every finished test, all skills together.
func (s Stats) Attempts() int {
	return s.Reading + s.Listening + s.Writing + s.Speaking
}

// Sort is what the board is ranked by.
type Sort string

const (
	SortXP       Sort = "xp"
	SortAttempts Sort = "attempts"
)

// Entry is a ranked row as the API serves it.
type Entry struct {
	Rank           int     `json:"rank"`
	UserID         uint64  `json:"user_id"`
	Name           string  `json:"name"`
	ImageURL       string  `json:"image_url"`
	EquippedAvatar *string `json:"equipped_avatar"`
	EquippedFrame  *string `json:"equipped_frame"`
	Level          int     `json:"level"`
	XP             int     `json:"xp"`
	Attempts       int     `json:"attempts"`
	Skills         Skills  `json:"skills"`
	IsMe           bool    `json:"is_me"`
}

type Skills struct {
	Reading   int `json:"reading"`
	Listening int `json:"listening"`
	Writing   int `json:"writing"`
	Speaking  int `json:"speaking"`
}

type Board struct {
	Sort    Sort    `json:"sort"`
	Entries []Entry `json:"entries"`
	// Me is the caller's own row, set when they're ranked but outside the
	// entries shown, so the page can still say where they stand.
	Me *Entry `json:"me"`
	// Total is how many learners are ranked (anyone with XP or a test done).
	Total int `json:"total"`
}
