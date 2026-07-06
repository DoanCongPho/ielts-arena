package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	MountRoutes(r *mux.Router)
	RegisterUser(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	LoginUser(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
}

type service struct {
	users Repository
}

func NewService(users Repository) Service {
	return &service{users: users}
}

func (s *service) MountRoutes(r *mux.Router) {
	r.HandleFunc("/auth/register", s.registerHandler).Methods(http.MethodPost)
	r.HandleFunc("/auth/login", s.loginHandler).Methods(http.MethodPost)
	r.HandleFunc("/auth/refresh", s.refreshHandler).Methods(http.MethodPost)
}

func (s *service) LoginUser(ctx context.Context, req LoginRequest) (*AuthResponse, error) {

	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}
	accessToken, err := GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}
	refreshToken, err := GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	claims, err := VerifyToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}
	if claims.Type != TokenTypeRefresh {
		return nil, errors.New("token is not a refresh token")
	}

	user, err := s.users.FindByEmail(ctx, claims.Email)
	if err != nil {
		return nil, errors.New("user not found")
	}

	accessToken, err := GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}
	newRefreshToken, err := GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *service) RegisterUser(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost, // Cost factor (mặc định = 10)
	)
	if err != nil {
		return nil, err
	}
	newUser, err := s.users.CreateUser(ctx, &User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Level:        1,
		XP:           0,
		RankScore:    0,
	})
	if err != nil {
		return nil, err
	}
	accessToken, err := GenerateAccessToken(newUser)
	if err != nil {
		return nil, err
	}
	refreshToken, err := GenerateRefreshToken(newUser)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         newUser,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
