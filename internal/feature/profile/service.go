// Package profile is the read/compose layer for the "who am I and how am
// I doing" screen. It owns no table and no rules of its own: identity
// comes from platform/auth, game progress from feature/progression, and
// this package's job is to join the two into one HTTP response.
package profile

import (
	"context"

	"github/DoanCongPho/game-arena/internal/feature/progression"
	"github/DoanCongPho/game-arena/internal/platform/auth"
)

type Service interface {
	GetProfile(ctx context.Context, userID uint64) (*ProfileResponse, error)
	SetEquippedFrame(ctx context.Context, userID uint64, frameLevel int) (*ProfileResponse, error)
}

type service struct {
	users       auth.Repository
	progression progression.Service
}

func NewService(users auth.Repository, prog progression.Service) Service {
	return &service{users: users, progression: prog}
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
	return newProfileResponse(user, prog), nil
}

func (s *service) SetEquippedFrame(ctx context.Context, userID uint64, frameLevel int) (*ProfileResponse, error) {
	// The unlock rule belongs to progression, not here — this package
	// only decides how to present the result.
	prog, err := s.progression.SetEquippedFrame(ctx, userID, frameLevel)
	if err != nil {
		return nil, err
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return newProfileResponse(user, prog), nil
}
