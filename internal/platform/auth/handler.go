package auth

import (
	"encoding/json"
	"github/DoanCongPho/game-arena/internal/platform/httpx"
	"net/http"
)

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
func (s *service) refreshHandler(w http.ResponseWriter, r *http.Request) {
	var body RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", "invalid JSON body")
		return
	}
	if err := body.Validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", err.Error())
		return
	}
	res, err := s.RefreshToken(r.Context(), body.RefreshToken)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_token", err.Error())
		return
	}
	httpx.WriteSuccess(w, res)
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
