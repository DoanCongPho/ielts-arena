package ielts_test

import (
	"encoding/json"
	"reflect"
	"testing"
)

func mustAnswerValue(t *testing.T, raw string) AnswerValue {
	t.Helper()
	var a AnswerValue
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("unmarshal %q: %v", raw, err)
	}
	return a
}

func TestAnswerValue_IsEmpty(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"unset", "", true},
		{"explicit null", "null", true},
		{"scalar string", `"true"`, false},
		{"array", `["A","B"]`, false},
		{"whitespace null", "  null  ", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var a AnswerValue
			if tc.raw != "" {
				a = mustAnswerValue(t, tc.raw)
			}
			if got := a.IsEmpty(); got != tc.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAnswerValue_Strings(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"scalar", `"paris"`, []string{"paris"}},
		{"array", `["A","C"]`, []string{"A", "C"}},
		{"null", "null", nil},
		{"empty array", `[]`, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := mustAnswerValue(t, tc.raw)
			got := a.Strings()
			if tc.want == nil {
				if got != nil {
					t.Errorf("Strings() = %v, want nil", got)
				}
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Strings() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAnswerValue_MarshalJSON_RoundTrip(t *testing.T) {
	a := mustAnswerValue(t, `["A","B"]`)
	out, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `["A","B"]` {
		t.Errorf("marshal = %s, want [\"A\",\"B\"]", out)
	}

	var zero AnswerValue
	out, err = json.Marshal(zero)
	if err != nil {
		t.Fatalf("marshal zero value: %v", err)
	}
	if string(out) != "null" {
		t.Errorf("marshal zero value = %s, want null", out)
	}
}
