package shop

import "errors"

var (
	ErrNotFound       = errors.New("user not found")
	ErrUnknownItem    = errors.New("unknown item")
	ErrInsufficientXP = errors.New("not enough XP")
	ErrAlreadyOwned   = errors.New("item already owned")
	ErrNotOwned       = errors.New("item not owned")
	ErrOutOfSeason    = errors.New("item is not on sale now")
	ErrWrongSlot      = errors.New("item does not go in that slot")
)

// Wallet is what a user can spend and what they're wearing. XP is
// lifetime XP (it decides the level and only ever grows); Spent is what
// purchases have taken out of it.
type Wallet struct {
	XP             int
	Spent          int
	EquippedAvatar *string
	EquippedFrame  *string
}

// Balance is the XP left to spend.
func (w *Wallet) Balance() int {
	if b := w.XP - w.Spent; b > 0 {
		return b
	}
	return 0
}
