// Package templates embeds the auth HTML entry pages (/login, /pending) and
// their compiled CSS. We keep templates here (not under a global views dir)
// because they are owned by platform/auth and serve only the pre-SPA flow.
package templates

import "embed"

//go:embed *.html login.css
var FS embed.FS

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