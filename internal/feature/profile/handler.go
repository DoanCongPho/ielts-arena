package profile

import (
	"encoding/json"
	"errors"
	"net/http"

	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/httpx"

	"github.com/gorilla/mux"
)

// Handler is the HTTP transport for the profile feature.
type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) MountRoutes(r *mux.Router) {
	r.HandleFunc("/profile", h.getProfileHandler).Methods(http.MethodGet)
	r.HandleFunc("/profile/frame", h.setEquippedFrameHandler).Methods(http.MethodPut)
}

func (h *Handler) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}

	resp, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "profile.not_found", err.Error())
			return
		}
		httpx.WriteInternalError(w, "profile.get_profile", err)
		return
	}
	httpx.WriteSuccess(w, resp)
}

func (h *Handler) setEquippedFrameHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
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

	resp, err := h.svc.SetEquippedFrame(r.Context(), userID, body.FrameLevel)
	if err != nil {
		switch {
		case errors.Is(err, ErrFrameLocked):
			httpx.WriteError(w, http.StatusBadRequest, "profile.frame_locked", err.Error())
		case errors.Is(err, auth.ErrUserNotFound):
			httpx.WriteError(w, http.StatusNotFound, "profile.not_found", err.Error())
		default:
			httpx.WriteInternalError(w, "profile.set_equipped_frame", err)
		}
		return
	}
	httpx.WriteSuccess(w, resp)
}
