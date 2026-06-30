package ielts_test

import "encoding/json"

type SubmitRequest struct {
	TestID  uint64          `json:"test_id"`
	Payload json.RawMessage `json:"payload"`
}

type TestResponse struct {
	ID          uint64          `json:"id"`
	Skill       string          `json:"skill"`
	TaskType    string          `json:"task_type"`
	ContentData json.RawMessage `json:"content_data"`
	XPGain      int             `json:"xp_gain"`
}

type SubmissionResponse struct {
	ID          uint64          `json:"id"`
	TestID      uint64          `json:"test_id"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	SubmittedAt string          `json:"submitted_at"`
}

func (r *SubmitRequest) Validate() error {
	// TODO: TestID phải > 0
	// TODO: Payload không được rỗng (len(r.Payload) == 0)
	return nil
}
