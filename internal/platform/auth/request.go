package auth

import "errors"

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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

func (r *LoginRequest) Validate() error {
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
