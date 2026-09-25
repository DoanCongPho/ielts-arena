package ielts_test

import (
	"encoding/json"
	"testing"
)

func TestListTestRequest_Offset(t *testing.T) {
	cases := []struct {
		page int
		want int
	}{
		{0, 0},
		{1, 0},
		{2, 12},
		{5, 48},
	}
	for _, tc := range cases {
		r := ListTestRequest{Page: tc.page}
		if got := r.Offset(); got != tc.want {
			t.Errorf("Page=%d: Offset() = %d, want %d", tc.page, got, tc.want)
		}
		if got := r.Limit(); got != defaultPageSize {
			t.Errorf("Limit() = %d, want %d", got, defaultPageSize)
		}
	}
}

func TestListSubmissionRequest_Offset(t *testing.T) {
	cases := []struct {
		page int
		want int
	}{
		{0, 0},
		{1, 0},
		{3, 24},
	}
	for _, tc := range cases {
		r := ListSubmissionRequest{Page: tc.page}
		if got := r.Offset(); got != tc.want {
			t.Errorf("Page=%d: Offset() = %d, want %d", tc.page, got, tc.want)
		}
	}
}

func TestSubmitRequest_Validate(t *testing.T) {
	cases := []struct {
		name    string
		req     SubmitRequest
		wantErr bool
	}{
		{"missing test_id", SubmitRequest{Payload: json.RawMessage(`{}`)}, true},
		{"missing payload", SubmitRequest{TestID: 1}, true},
		{"valid", SubmitRequest{TestID: 1, Payload: json.RawMessage(`{"text":"hi"}`)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestCreateTestRequest_Validate(t *testing.T) {
	validWriting := json.RawMessage(`{"prompt":"Describe the chart."}`)

	cases := []struct {
		name    string
		req     CreateTestRequest
		wantErr bool
	}{
		{
			name:    "invalid skill",
			req:     CreateTestRequest{Skill: "vocabulary", TaskType: "task1", ContentData: validWriting},
			wantErr: true,
		},
		{
			name:    "missing task_type",
			req:     CreateTestRequest{Skill: "writing", ContentData: validWriting},
			wantErr: true,
		},
		{
			name:    "missing content_data",
			req:     CreateTestRequest{Skill: "writing", TaskType: "task1"},
			wantErr: true,
		},
		{
			name:    "negative xp_gain",
			req:     CreateTestRequest{Skill: "writing", TaskType: "task1", ContentData: validWriting, XPGain: -1},
			wantErr: true,
		},
		{
			name:    "content_data fails skill-specific validation",
			req:     CreateTestRequest{Skill: "reading", TaskType: "task1", ContentData: json.RawMessage(`{}`)},
			wantErr: true,
		},
		{
			name:    "series without volume",
			req:     CreateTestRequest{Skill: "writing", TaskType: "task1", Series: "cambridge", TestNumber: 1, ContentData: validWriting},
			wantErr: true,
		},
		{
			name:    "series without test_number",
			req:     CreateTestRequest{Skill: "writing", TaskType: "task1", Series: "cambridge", Volume: 20, ContentData: validWriting},
			wantErr: true,
		},
		{
			name:    "valid in a series",
			req:     CreateTestRequest{Skill: "writing", TaskType: "task1", Series: "cambridge", Volume: 20, TestNumber: 1, ContentData: validWriting},
			wantErr: false,
		},
		{
			name:    "valid",
			req:     CreateTestRequest{Skill: "writing", TaskType: "task2", ContentData: validWriting, XPGain: 10},
			wantErr: false,
		},
		{
			name:    "writing task_type other than task1/task2",
			req:     CreateTestRequest{Skill: "writing", TaskType: "essay", ContentData: validWriting},
			wantErr: true,
		},
		{
			name:    "writing chart as an asset path",
			req:     CreateTestRequest{Skill: "writing", TaskType: "task1", ContentData: json.RawMessage(`{"prompt":"Describe the chart.","image_url":"/assets/writing_charts/a.png"}`)},
			wantErr: false,
		},
		{
			name:    "writing chart as a bare relative path",
			req:     CreateTestRequest{Skill: "writing", TaskType: "task1", ContentData: json.RawMessage(`{"prompt":"Describe the chart.","image_url":"charts/a.png"}`)},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
