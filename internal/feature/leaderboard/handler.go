package leaderboard

import (
	"net/http"
	"strconv"

	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/httpx"

	"github.com/gorilla/mux"
)

// Handler serves GET /api/leaderboard?sort=xp|attempts&limit=N. Mounted
// on the /api subrouter, so RequireAuth already applies.
type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) MountRoutes(r *mux.Router) {
	r.HandleFunc("/leaderboard", h.getBoard).Methods(http.MethodGet)
}

func (h *Handler) getBoard(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	board, err := h.svc.Board(r.Context(), Sort(r.URL.Query().Get("sort")), userID, limit)
	if err != nil {
		httpx.WriteInternalError(w, "leaderboard.get", err)
		return
	}
	httpx.WriteSuccess(w, board)
}
