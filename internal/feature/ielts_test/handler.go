package ielts_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/httpx"
	"github/DoanCongPho/game-arena/internal/platform/middleware"

	"github.com/gorilla/mux"
)

// Handler is the HTTP transport for this feature. It owns everything that
// knows about routers, status codes and JSON bodies; Service below it owns
// the business rules and knows nothing about HTTP. Keeping the two apart is
// what lets Service be tested without a router and swapped behind a
// different transport without touching its contract.
type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) MountRoutes(r *mux.Router) {
	r.HandleFunc("/tests", h.listTestsHandler).Methods(http.MethodGet)
	// Creating tests is admin-only — RequireAuth (already applied to the
	// whole /api subrouter, see cmd/api/main.go) has run by this point, so
	// RequireAdmin only needs to check the role already on the request
	// context, not re-verify the token.
	r.Handle("/tests", middleware.RequireAdmin(http.HandlerFunc(h.createTestHandler))).Methods(http.MethodPost)
	r.HandleFunc("/tests/{id}", h.getTestHandler).Methods(http.MethodGet)
	r.HandleFunc("/submissions", h.submitAnswerHandler).Methods(http.MethodPost)
	r.HandleFunc("/submissions", h.listSubmissionsHandler).Methods(http.MethodGet)
	r.HandleFunc("/submissions/{id}", h.getSubmissionHandler).Methods(http.MethodGet)
	r.HandleFunc("/submissions/{id}/score", h.getScoreHandler).Methods(http.MethodGet)
}

func (h *Handler) listTestsHandler(w http.ResponseWriter, r *http.Request) {
	skill := r.URL.Query().Get("skill")
	if skill == "" {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "skill is required")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))

	resp, err := h.svc.GetListTest(r.Context(), skill, ListTestRequest{Page: page})
	if err != nil {
		httpx.WriteInternalError(w, "ielts_test.list_tests", err)
		return
	}
	httpx.WriteSuccess(w, resp)
}

func (h *Handler) getTestHandler(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.IDFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid test id")
		return
	}

	test, err := h.svc.GetTest(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTestNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "ielts_test.test_not_found", err.Error())
			return
		}
		httpx.WriteInternalError(w, "ielts_test.get_test", err)
		return
	}
	httpx.WriteSuccess(w, test)
}

func (h *Handler) createTestHandler(w http.ResponseWriter, r *http.Request) {
	var body CreateTestRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid JSON body")
		return
	}
	if err := body.Validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", err.Error())
		return
	}

	created, err := h.svc.PostTest(r.Context(), Test{
		Skill:        body.Skill,
		TaskType:     body.TaskType,
		ContentData:  body.ContentData,
		ThumbnailURL: body.ThumbnailURL,
		Source:       body.Source,
		IsCurrent:    body.IsCurrent,
		XPGain:       body.XPGain,
	})
	if err != nil {
		httpx.WriteInternalError(w, "ielts_test.create_test", err)
		return
	}

	content, err := publicContentData(created.Skill, created.ContentData)
	if err != nil {
		httpx.WriteInternalError(w, "ielts_test.create_test", err)
		return
	}
	httpx.WriteSuccess(w, newTestResponse(created, content))
}

func (h *Handler) submitAnswerHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}

	var body SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid JSON body")
		return
	}
	if err := body.Validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", err.Error())
		return
	}

	sub, err := h.svc.SubmitAnswer(r.Context(), userID, body)
	if err != nil {
		if errors.Is(err, ErrTestNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "ielts_test.test_not_found", err.Error())
			return
		}
		httpx.WriteInternalError(w, "ielts_test.submit_answer", err)
		return
	}
	// 202: the submission is persisted and queued, but grading happens in
	// the background worker — the client polls GET /submissions/{id} until
	// status leaves "pending"/"grading".
	httpx.WriteStatus(w, http.StatusAccepted, newSubmissionResponse(sub))
}

func (h *Handler) listSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))

	resp, err := h.svc.GetListSubmission(r.Context(), userID, ListSubmissionRequest{Page: page})
	if err != nil {
		httpx.WriteInternalError(w, "ielts_test.list_submissions", err)
		return
	}
	httpx.WriteSuccess(w, resp)
}

func (h *Handler) getSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	id, err := httpx.IDFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid submission id")
		return
	}

	sub, err := h.svc.GetSubmissionByID(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrSubmissionNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "ielts_test.submission_not_found", err.Error())
			return
		}
		httpx.WriteInternalError(w, "ielts_test.get_submission", err)
		return
	}
	httpx.WriteSuccess(w, newSubmissionResponse(sub))
}

func (h *Handler) getScoreHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.CurrentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	id, err := httpx.IDFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid submission id")
		return
	}

	score, err := h.svc.GetScore(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrSubmissionNotFound) || errors.Is(err, ErrScoreNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "ielts_test.score_not_found", err.Error())
			return
		}
		httpx.WriteInternalError(w, "ielts_test.get_score", err)
		return
	}
	httpx.WriteSuccess(w, newScoreResponse(score))
}
