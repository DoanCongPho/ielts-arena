package auth

import (
	"errors"
	"net/mail"
	"strings"
)

// gmailSuffix is the only accepted domain. Sign-ups are Gmail-only, which
// keeps a typo like "gmail.con" — which silently created a second,
// unreachable account — from ever reaching the users table.
const gmailSuffix = "@gmail.com"

// normalizeEmail trims and lowercases, so "  Abc@Gmail.com " and
// "abc@gmail.com" can't become two accounts (the unique index is on the
// raw column and would see them as different).
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validateGmail returns the normalized address, or an error naming what
// is wrong in Vietnamese — these strings reach the sign-up form as-is.
func validateGmail(email string) (string, error) {
	email = normalizeEmail(email)
	if email == "" {
		return "", errors.New("email required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", errors.New("email không hợp lệ")
	}
	if !strings.HasSuffix(email, gmailSuffix) {
		return "", errors.New("chỉ hỗ trợ email @gmail.com")
	}
	if len(email) == len(gmailSuffix) {
		return "", errors.New("email không hợp lệ")
	}
	return email, nil
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Response data
type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (r *RegisterRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name required")
	}
	email, err := validateGmail(r.Email)
	if err != nil {
		return err
	}
	r.Email = email
	if r.Password == "" {
		return errors.New("password required")
	}
	if len(r.Password) < 8 {
		return errors.New("password must be 8+ chars")
	}
	return nil
}

func (r *RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return errors.New("refresh_token required")
	}
	return nil
}

func (r *LoginRequest) Validate() error {
	r.Email = normalizeEmail(r.Email)
	if r.Email == "" {
		return errors.New("email required")
	}
	if r.Password == "" {
		return errors.New("password required")
	}
	if len(r.Password) < 8 {
		return errors.New("password must be 8+ chars")
	}
	return nil
}
