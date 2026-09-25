package ielts_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/httpx"

	"github.com/gorilla/mux"
)

// SpeakingHandler is the HTTP transport for SpeakingService.
type SpeakingHandler struct {
	svc *SpeakingService
}

func NewSpeakingHandler(svc *SpeakingService) *SpeakingHandler {
	return &SpeakingHandler{svc: svc}
}

func (h *SpeakingHandler) MountRoutes(r *mux.Router) {
	r.HandleFunc("/speaking/uploads", h.uploads).Methods(http.MethodPost)
	r.HandleFunc("/speaking/tests/{id}/script", h.script).Methods(http.MethodGet)
	r.HandleFunc("/speaking/custom", h.listCustom).Methods(http.MethodGet)
	r.HandleFunc("/speaking/custom", h.createCustom).Methods(http.MethodPost)
	r.HandleFunc("/speaking/custom/generate", h.generate).Methods(http.MethodPost)
	r.HandleFunc("/speaking/custom/{id}", h.updateCustom).Methods(http.MethodPut)
	r.HandleFunc("/speaking/custom/{id}", h.deleteCustom).Methods(http.MethodDelete)
	r.HandleFunc("/speaking/submissions/{id}/recordings", h.recordings).Methods(http.MethodGet)
}

func (h *SpeakingHandler) uploads(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Count int    `json:"count"`
		Ext   string `json:"ext"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid JSON body")
		return
	}
	slots, err := h.svc.UploadSlots(userID, body.Count, body.Ext)
	if err != nil {
		writeSpeakingError(w, "ielts_test.speaking_uploads", err)
		return
	}
	httpx.WriteSuccess(w, slots)
}

func (h *SpeakingHandler) script(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id, err := httpx.IDFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid test id")
		return
	}
	script, err := h.svc.Script(r.Context(), userID, id)
	if err != nil {
		writeSpeakingError(w, "ielts_test.speaking_script", err)
		return
	}
	httpx.WriteSuccess(w, script)
}

func (h *SpeakingHandler) listCustom(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	resp, err := h.svc.ListCustom(r.Context(), userID, page)
	if err != nil {
		httpx.WriteInternalError(w, "ielts_test.speaking_list_custom", err)
		return
	}
	httpx.WriteSuccess(w, resp)
}

func (h *SpeakingHandler) createCustom(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body ComposeSpeakingRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid JSON body")
		return
	}
	res, err := h.svc.CreateCustom(r.Context(), userID, body)
	if err != nil {
		writeSpeakingError(w, "ielts_test.speaking_create_custom", err)
		return
	}
	httpx.WriteStatus(w, http.StatusCreated, res)
}

func (h *SpeakingHandler) updateCustom(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id, err := httpx.IDFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid test id")
		return
	}
	var body ComposeSpeakingRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid JSON body")
		return
	}
	res, err := h.svc.UpdateCustom(r.Context(), userID, id, body)
	if err != nil {
		writeSpeakingError(w, "ielts_test.speaking_update_custom", err)
		return
	}
	httpx.WriteSuccess(w, res)
}

func (h *SpeakingHandler) deleteCustom(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id, err := httpx.IDFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid test id")
		return
	}
	if err := h.svc.DeleteCustom(r.Context(), userID, id); err != nil {
		writeSpeakingError(w, "ielts_test.speaking_delete_custom", err)
		return
	}
	httpx.WriteSuccess(w, map[string]bool{"deleted": true})
}

func (h *SpeakingHandler) generate(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUser(w, r); !ok {
		return
	}
	var body GeneratePartsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid JSON body")
		return
	}
	content, err := h.svc.GenerateParts(r.Context(), body)
	if err != nil {
		writeSpeakingError(w, "ielts_test.speaking_generate", err)
		return
	}
	httpx.WriteSuccess(w, content)
}

func (h *SpeakingHandler) recordings(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id, err := httpx.IDFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid submission id")
		return
	}
	links, err := h.svc.Recordings(r.Context(), userID, id)
	if err != nil {
		writeSpeakingError(w, "ielts_test.speaking_recordings", err)
		return
	}
	httpx.WriteSuccess(w, links)
}

func requireUser(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
	}
	return userID, ok
}

func writeSpeakingError(w http.ResponseWriter, code string, err error) {
	switch {
	case errors.Is(err, ErrTestNotFound):
		httpx.WriteError(w, http.StatusNotFound, "ielts_test.test_not_found", err.Error())
	case errors.Is(err, ErrSubmissionNotFound):
		httpx.WriteError(w, http.StatusNotFound, "ielts_test.submission_not_found", err.Error())
	case errors.Is(err, ErrInvalidSpeakingContent):
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", err.Error())
	case errors.Is(err, ErrContentFlagged):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "ielts_test.content_flagged", err.Error())
	case errors.Is(err, ErrTestHasAttempts):
		httpx.WriteError(w, http.StatusConflict, "ielts_test.test_has_attempts", err.Error())
	case errors.Is(err, ErrTooManyCustomTests):
		httpx.WriteError(w, http.StatusConflict, "ielts_test.too_many_custom_tests", err.Error())
	default:
		httpx.WriteInternalError(w, code, err)
	}
}
