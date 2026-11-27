package music

import (
	"encoding/json"
	"errors"
	"net/http"

	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/httpx"

	"github.com/gorilla/mux"
)

// Handler serves the learner's own music library:
//
//	GET    /api/music           the caller's tracks, with play links
//	POST   /api/music/uploads   {filename, size} -> a slot to PUT the file to
//	POST   /api/music           {key, filename, title} once the PUT is done
//	DELETE /api/music/{id}      remove one of the caller's tracks
//
// Mounted on the /api subrouter, so RequireAuth already applies.
type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) MountRoutes(r *mux.Router) {
	r.HandleFunc("/music", h.list).Methods(http.MethodGet)
	r.HandleFunc("/music/uploads", h.requestUpload).Methods(http.MethodPost)
	r.HandleFunc("/music", h.confirm).Methods(http.MethodPost)
	r.HandleFunc("/music/{id:[0-9]+}", h.remove).Methods(http.MethodDelete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	lib, err := h.svc.Library(r.Context(), userID)
	if err != nil {
		writeMusicError(w, "music.list", err)
		return
	}
	httpx.WriteSuccess(w, lib)
}

func (h *Handler) requestUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	var body struct {
		Filename string `json:"filename"`
		Size     int    `json:"size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "music.invalid_request", "filename and size are required")
		return
	}
	slot, err := h.svc.RequestUpload(r.Context(), userID, body.Filename, body.Size)
	if err != nil {
		writeMusicError(w, "music.request_upload", err)
		return
	}
	httpx.WriteSuccess(w, slot)
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	var body struct {
		Key      string `json:"key"`
		Filename string `json:"filename"`
		Title    string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
		httpx.WriteError(w, http.StatusBadRequest, "music.invalid_request", "key is required")
		return
	}
	track, err := h.svc.Confirm(r.Context(), userID, body.Key, body.Filename, body.Title)
	if err != nil {
		writeMusicError(w, "music.confirm", err)
		return
	}
	httpx.WriteStatus(w, http.StatusCreated, track)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	id, err := httpx.IDFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "music.invalid_request", "invalid track id")
		return
	}
	if err := h.svc.Delete(r.Context(), userID, id); err != nil {
		writeMusicError(w, "music.delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeMusicError(w http.ResponseWriter, code string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "music.not_found", "Không tìm thấy bài này.")
	case errors.Is(err, ErrTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "music.too_large", "File nhạc tối đa 15 MB.")
	case errors.Is(err, ErrTooMany):
		httpx.WriteError(w, http.StatusConflict, "music.too_many", "Bạn đã có 20 bài — xoá bớt rồi tải thêm nhé.")
	case errors.Is(err, ErrNotAudio):
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "music.not_audio", "Chỉ nhận file nhạc: mp3, m4a, aac, ogg, opus, wav, webm, flac.")
	case errors.Is(err, ErrEmptyUpload):
		httpx.WriteError(w, http.StatusBadRequest, "music.empty", "File trống.")
	default:
		httpx.WriteInternalError(w, code, err)
	}
}
