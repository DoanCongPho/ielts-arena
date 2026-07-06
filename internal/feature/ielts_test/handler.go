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

func (s *service) MountRoutes(r *mux.Router) {
	r.HandleFunc("/tests", s.listTestsHandler).Methods(http.MethodGet)
	r.HandleFunc("/tests", s.createTestHandler).Methods(http.MethodPost)
	r.HandleFunc("/tests/{id}", s.getTestHandler).Methods(http.MethodGet)
	r.HandleFunc("/submissions", s.submitAnswerHandler).Methods(http.MethodPost)
	r.HandleFunc("/submissions", s.listSubmissionsHandler).Methods(http.MethodGet)
	r.HandleFunc("/submissions/{id}", s.getSubmissionHandler).Methods(http.MethodGet)
	r.HandleFunc("/submissions/{id}/score", s.getScoreHandler).Methods(http.MethodGet)
}

func currentUserID(r *http.Request) (uint64, bool) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		return 0, false
	}
	return user.ID, true
}

func idFromPath(r *http.Request) (uint64, error) {
	return strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
}

func (s *service) listTestsHandler(w http.ResponseWriter, r *http.Request) {
	skill := r.URL.Query().Get("skill")
	if skill == "" {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "skill is required")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))

	resp, err := s.GetListTest(r.Context(), skill, ListTestRequest{Page: page})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "ielts_test.list_tests", err.Error())
		return
	}
	httpx.WriteSuccess(w, resp)
}

func (s *service) getTestHandler(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid test id")
		return
	}

	test, err := s.GetTest(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTestNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "ielts_test.test_not_found", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "ielts_test.get_test", err.Error())
		return
	}
	httpx.WriteSuccess(w, test)
}

func (s *service) createTestHandler(w http.ResponseWriter, r *http.Request) {
	var body CreateTestRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid JSON body")
		return
	}
	if err := body.Validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", err.Error())
		return
	}

	created, err := s.PostTest(r.Context(), Test{
		Skill:       body.Skill,
		TaskType:    body.TaskType,
		ContentData: body.ContentData,
		Source:      body.Source,
		IsCurrent:   body.IsCurrent,
		XPGain:      body.XPGain,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "ielts_test.create_test", err.Error())
		return
	}

	content, err := publicContentData(created.Skill, created.ContentData)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "ielts_test.create_test", err.Error())
		return
	}
	httpx.WriteSuccess(w, TestResponse{
		ID:          created.ID,
		Skill:       created.Skill,
		TaskType:    created.TaskType,
		ContentData: content,
		XPGain:      created.XPGain,
	})
}

func (s *service) submitAnswerHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
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

	sub, err := s.SubmitAnswer(r.Context(), userID, body)
	if err != nil {
		if errors.Is(err, ErrTestNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "ielts_test.test_not_found", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "ielts_test.submit_answer", err.Error())
		return
	}
	httpx.WriteSuccess(w, newSubmissionResponse(sub))
}

func (s *service) listSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))

	resp, err := s.GetListSubmission(r.Context(), userID, ListSubmissionRequest{Page: page})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "ielts_test.list_submissions", err.Error())
		return
	}
	httpx.WriteSuccess(w, resp)
}

func (s *service) getSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	id, err := idFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid submission id")
		return
	}

	sub, err := s.GetSubmissionByID(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrSubmissionNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "ielts_test.submission_not_found", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "ielts_test.get_submission", err.Error())
		return
	}
	httpx.WriteSuccess(w, newSubmissionResponse(sub))
}

func (s *service) getScoreHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "auth.invalid_credentials", "missing authenticated user")
		return
	}
	id, err := idFromPath(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ielts_test.invalid_input", "invalid submission id")
		return
	}

	score, err := s.GetScore(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrSubmissionNotFound) || errors.Is(err, ErrScoreNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "ielts_test.score_not_found", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "ielts_test.get_score", err.Error())
		return
	}
	httpx.WriteSuccess(w, newScoreResponse(score))
}
