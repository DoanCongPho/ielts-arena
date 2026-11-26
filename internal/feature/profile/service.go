// Package profile is the read/compose layer for the "who am I and how am
// I doing" screen. It owns no table and no rules of its own: identity
// comes from platform/auth, game progress from feature/progression, and
// this package's job is to join the two into one HTTP response.
package profile

import (
	"context"

	"github/DoanCongPho/game-arena/internal/feature/progression"
	"github/DoanCongPho/game-arena/internal/feature/shop"
	"github/DoanCongPho/game-arena/internal/platform/auth"
)

// WalletReader is the slice of the shop the profile shows: the XP left to
// spend and the avatar and frame being worn.
type WalletReader interface {
	Wallet(ctx context.Context, userID uint64) (*shop.Wallet, error)
}

type Service interface {
	GetProfile(ctx context.Context, userID uint64) (*ProfileResponse, error)
}

type service struct {
	users       auth.Repository
	progression progression.Service
	wallets     WalletReader
}

// NewService composes the profile. wallets may be nil, in which case the
// shop fields are left at their zero values.
func NewService(users auth.Repository, prog progression.Service, wallets WalletReader) Service {
	return &service{users: users, progression: prog, wallets: wallets}
}

func (s *service) GetProfile(ctx context.Context, userID uint64) (*ProfileResponse, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	prog, err := s.progression.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := newProfileResponse(user, prog)
	if s.wallets != nil {
		w, err := s.wallets.Wallet(ctx, userID)
		if err != nil {
			return nil, err
		}
		resp.XPBalance = w.Balance()
		resp.EquippedAvatar = w.EquippedAvatar
		resp.EquippedFrame = w.EquippedFrame
	}
	return resp, nil
}
