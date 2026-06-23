package auth

import (
	"context"
	"encoding/json"
	"errors"
	"github/DoanCongPho/game-arena/internal/platform/httpx"
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	MountRoutes(r *mux.Router)
	RegisterUser(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	LoginUser(ctx context.Context, req LoginRequest) (*AuthResponse, error)
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
}

func (s *service) registerHandler(w http.ResponseWriter, r *http.Request) {
	var body RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", "invalid JSON body")
		return
	}
	if err := body.Validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", err.Error())
		return
	}
	user, err := s.RegisterUser(r.Context(), body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.create_user", err.Error())
		return
	}
	httpx.WriteSuccess(w, user)
}

func (s *service) loginHandler(w http.ResponseWriter, r *http.Request) {
	var body LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", "invalid JSON body")
		return
	}
	if err := body.Validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", err.Error())
		return
	}
	res, err := s.LoginUser(r.Context(), body)

	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", err.Error())
		return
	}
	httpx.WriteSuccess(w, res)

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
