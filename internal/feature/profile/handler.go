package profile

import (
	"errors"
	"net/http"

	"github/DoanCongPho/game-arena/internal/feature/progression"
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
}

func (h *Handler) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}

	resp, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) || errors.Is(err, progression.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "profile.not_found", err.Error())
			return
		}
		httpx.WriteInternalError(w, "profile.get_profile", err)
		return
	}
	httpx.WriteSuccess(w, resp)
}
