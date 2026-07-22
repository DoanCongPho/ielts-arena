package profile

import (
	"encoding/json"
	"errors"
	"net/http"

	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/httpx"

	"github.com/gorilla/mux"
)

func (s *service) MountRoutes(r *mux.Router) {
	r.HandleFunc("/profile", s.getProfileHandler).Methods(http.MethodGet)
	r.HandleFunc("/profile/frame", s.setEquippedFrameHandler).Methods(http.MethodPut)
}

func currentUserID(r *http.Request) (uint64, bool) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		return 0, false
	}
	return user.ID, true
}

func (s *service) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}

	resp, err := s.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "profile.not_found", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "profile.get_profile", err.Error())
		return
	}
	httpx.WriteSuccess(w, resp)
}

func (s *service) setEquippedFrameHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}

	var body SetEquippedFrameRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "profile.invalid_input", "invalid JSON body")
		return
	}
	if err := body.Validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "profile.invalid_input", err.Error())
		return
	}

	resp, err := s.SetEquippedFrame(r.Context(), userID, body.FrameLevel)
	if err != nil {
		switch {
		case errors.Is(err, ErrFrameLocked):
			httpx.WriteError(w, http.StatusBadRequest, "profile.frame_locked", err.Error())
		case errors.Is(err, auth.ErrUserNotFound):
			httpx.WriteError(w, http.StatusNotFound, "profile.not_found", err.Error())
		default:
			httpx.WriteError(w, http.StatusInternalServerError, "profile.set_equipped_frame", err.Error())
		}
		return
	}
	httpx.WriteSuccess(w, resp)
}
