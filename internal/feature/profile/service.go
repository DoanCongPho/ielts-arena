// Package profile exposes the current user's game-progress view (level, XP,
// avatar frame) and lets them pick which unlocked frame to display. It has
// no table of its own — the users row is owned by auth.Repository, so this
// package is a thin read/business-rule layer over that repository rather
// than a second, parallel gateway to the same data.
package profile

import (
	"context"

	"github/DoanCongPho/game-arena/internal/platform/auth"

	"github.com/gorilla/mux"
)

type Service interface {
	MountRoutes(r *mux.Router)
	GetProfile(ctx context.Context, userID uint64) (*ProfileResponse, error)
	SetEquippedFrame(ctx context.Context, userID uint64, frameLevel int) (*ProfileResponse, error)
}

type service struct {
	users auth.Repository
}

func NewService(users auth.Repository) Service {
	return &service{users: users}
}

func (s *service) GetProfile(ctx context.Context, userID uint64) (*ProfileResponse, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return newProfileResponse(u), nil
}

func (s *service) SetEquippedFrame(ctx context.Context, userID uint64, frameLevel int) (*ProfileResponse, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if frameLevel < 1 || frameLevel > u.Level {
		return nil, ErrFrameLocked
	}
	if err := s.users.UpdateEquippedFrame(ctx, userID, frameLevel); err != nil {
		return nil, err
	}
	u.EquippedFrameLevel = &frameLevel
	return newProfileResponse(u), nil
}
