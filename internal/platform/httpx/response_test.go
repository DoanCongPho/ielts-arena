package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewPagination(t *testing.T) {
	tests := []struct {
		name                     string
		total, page, limit       int
		wantTotalPages           int
		wantHasNext, wantHasPrev bool
	}{
		{"empty result set", 0, 1, 10, 0, false, false},
		{"single partial page", 7, 1, 10, 1, false, false},
		{"exactly one full page", 10, 1, 10, 1, false, false},
		{"first of three pages", 25, 1, 10, 3, true, false},
		{"middle of three pages", 25, 2, 10, 3, true, true},
		{"last of three pages", 25, 3, 10, 3, false, true},
		{"page past the end", 25, 9, 10, 3, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NewPagination(tc.total, tc.page, tc.limit)
			if got.TotalPages != tc.wantTotalPages {
				t.Errorf("TotalPages = %d, want %d", got.TotalPages, tc.wantTotalPages)
			}
			if got.HasNext != tc.wantHasNext {
				t.Errorf("HasNext = %v, want %v", got.HasNext, tc.wantHasNext)
			}
			if got.HasPrev != tc.wantHasPrev {
				t.Errorf("HasPrev = %v, want %v", got.HasPrev, tc.wantHasPrev)
			}
			if got.Total != tc.total {
				t.Errorf("Total = %d, want %d", got.Total, tc.total)
			}
		})
	}
}

func TestWriteSuccessWrapsInDataEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteSuccess(rec, map[string]string{"hello": "world"})

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Data["hello"] != "world" {
		t.Errorf("data = %v, want the payload under a \"data\" key", body.Data)
	}
}

func TestWriteStatusUsesGivenCode(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteStatus(rec, http.StatusAccepted, struct{}{})

	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
}

// The whole point of WriteInternalError is that the caller learns nothing
// about what actually broke.
func TestWriteInternalErrorHidesTheCause(t *testing.T) {
	rec := httptest.NewRecorder()
	secret := "Error 1146: Table 'ielts.submissions' doesn't exist"

	WriteInternalError(rec, "ielts_test.get_test", errorString(secret))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if strings.Contains(rec.Body.String(), secret) {
		t.Errorf("response leaked the underlying error: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "Table") || strings.Contains(rec.Body.String(), "1146") {
		t.Errorf("response leaked database details: %s", rec.Body.String())
	}

	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Code != "ielts_test.get_test" {
		t.Errorf("Code = %q, want the stable machine-readable code to survive", body.Code)
	}
	if body.Message == "" {
		t.Error("Message should still say something human-readable")
	}
}

type errorString string

func (e errorString) Error() string { return string(e) }
