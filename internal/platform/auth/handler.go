package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github/DoanCongPho/game-arena/internal/platform/httpx"

	"github.com/gorilla/mux"
)

// Handler is the HTTP transport for auth. Service holds the credential
// rules and token issuing; this type holds the routes, JSON decoding and
// status codes.
type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) MountRoutes(r *mux.Router) {
	r.HandleFunc("/auth/register", h.registerHandler).Methods(http.MethodPost)
	r.HandleFunc("/auth/login", h.loginHandler).Methods(http.MethodPost)
	r.HandleFunc("/auth/refresh", h.refreshHandler).Methods(http.MethodPost)
}

func (h *Handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	var body LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", "invalid JSON body")
		return
	}
	if err := body.Validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", err.Error())
		return
	}
	res, err := h.svc.LoginUser(r.Context(), body)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", err.Error())
		return
	}
	httpx.WriteSuccess(w, res)
}

func (h *Handler) refreshHandler(w http.ResponseWriter, r *http.Request) {
	var body RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", "invalid JSON body")
		return
	}
	if err := body.Validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", err.Error())
		return
	}
	res, err := h.svc.RefreshToken(r.Context(), body.RefreshToken)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_token", err.Error())
		return
	}
	httpx.WriteSuccess(w, res)
}

func (h *Handler) registerHandler(w http.ResponseWriter, r *http.Request) {
	var body RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", "invalid JSON body")
		return
	}
	if err := body.Validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "auth.invalid_input", err.Error())
		return
	}
	user, err := h.svc.RegisterUser(r.Context(), body)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			httpx.WriteError(w, http.StatusConflict, "auth.email_taken", "That email is already registered.")
			return
		}
		httpx.WriteInternalError(w, "auth.create_user", err)
		return
	}
	httpx.WriteSuccess(w, user)
}
