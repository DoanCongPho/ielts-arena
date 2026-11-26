package shop

import (
	"encoding/json"
	"errors"
	"net/http"

	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/httpx"

	"github.com/gorilla/mux"
)

// Handler is the HTTP transport for the shop. Mounted on the /api
// subrouter, so RequireAuth already applies.
type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) MountRoutes(r *mux.Router) {
	r.HandleFunc("/shop", h.getShop).Methods(http.MethodGet)
	r.HandleFunc("/shop/purchase", h.purchase).Methods(http.MethodPost)
	r.HandleFunc("/shop/equip", h.equip).Methods(http.MethodPost)
}

func (h *Handler) getShop(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	resp, err := h.svc.Shop(r.Context(), userID)
	if err != nil {
		writeShopError(w, "shop.get", err)
		return
	}
	httpx.WriteSuccess(w, resp)
}

func (h *Handler) purchase(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	var body struct {
		ItemID string `json:"item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ItemID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "shop.invalid_request", "item_id is required")
		return
	}
	resp, err := h.svc.Purchase(r.Context(), userID, body.ItemID)
	if err != nil {
		writeShopError(w, "shop.purchase", err)
		return
	}
	httpx.WriteSuccess(w, resp)
}

func (h *Handler) equip(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	// {"kind": "frame", "item_id": "frame-lights"}; an empty item_id takes
	// the frame off.
	var body struct {
		Kind   Kind   `json:"kind"`
		ItemID string `json:"item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Kind == "" {
		httpx.WriteError(w, http.StatusBadRequest, "shop.invalid_request", "kind is required")
		return
	}
	resp, err := h.svc.Equip(r.Context(), userID, body.Kind, body.ItemID)
	if err != nil {
		writeShopError(w, "shop.equip", err)
		return
	}
	httpx.WriteSuccess(w, resp)
}

func writeShopError(w http.ResponseWriter, code string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "shop.user_not_found", err.Error())
	case errors.Is(err, ErrUnknownItem):
		httpx.WriteError(w, http.StatusNotFound, "shop.unknown_item", "Không có vật phẩm này.")
	case errors.Is(err, ErrInsufficientXP):
		httpx.WriteError(w, http.StatusConflict, "shop.insufficient_xp", "Bạn chưa đủ XP để đổi vật phẩm này.")
	case errors.Is(err, ErrAlreadyOwned):
		httpx.WriteError(w, http.StatusConflict, "shop.already_owned", "Bạn đã có vật phẩm này rồi.")
	case errors.Is(err, ErrNotOwned):
		httpx.WriteError(w, http.StatusConflict, "shop.not_owned", "Bạn cần đổi vật phẩm này trước khi dùng.")
	case errors.Is(err, ErrOutOfSeason):
		httpx.WriteError(w, http.StatusConflict, "shop.out_of_season", "Vật phẩm này chỉ đổi được trong mùa Giáng sinh (1/12 – 6/1).")
	case errors.Is(err, ErrWrongSlot):
		httpx.WriteError(w, http.StatusBadRequest, "shop.wrong_slot", "Vật phẩm không dùng được ở vị trí này.")
	default:
		httpx.WriteInternalError(w, code, err)
	}
}
