package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SuccessReponse struct {
	Data interface{} `json:"data"`
}

type Pagination struct {
	Total      int  `json:"total"`
	Page       int  `json:"page"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

func NewPagination(total, page, limit int) Pagination {
	totalPages := (total + limit - 1) / limit
	return Pagination{
		Total:      total,
		Page:       page,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// WriteInternalError reports a server-side failure without handing the
// caller its details. The underlying error goes to the log, where it is
// useful; the client gets a stable code and a generic message.
//
// Passing err.Error() through to the response (as every 500 in this
// codebase used to) leaks SQL text, table and column names, driver
// details and upstream API error bodies to anyone who can trigger the
// failure — a free map of the system's internals.
func WriteInternalError(w http.ResponseWriter, code string, err error) {
	log.Printf("http 500 [%s]: %v", code, err)
	WriteError(w, http.StatusInternalServerError, code, "Something went wrong on our side. Please try again.")
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteStatus(w, http.StatusOK, data)
}

// WriteStatus is WriteSuccess with an explicit status code, for the cases
// where 200 is the wrong answer — e.g. 202 Accepted when work has been
// queued rather than completed.
func WriteStatus(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(SuccessReponse{Data: data})
}
