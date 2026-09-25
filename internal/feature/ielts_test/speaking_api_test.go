package ielts_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeStore is an in-memory object store.
type fakeStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newFakeStore() *fakeStore { return &fakeStore{objects: map[string][]byte{}} }

func (f *fakeStore) UploadURL(key string, _ time.Duration) (string, error) {
	return "https://bucket/" + key + "?put", nil
}
func (f *fakeStore) DownloadURL(key string, _ time.Duration) (string, error) {
	return "https://bucket/" + key + "?get", nil
}
func (f *fakeStore) Put(_ context.Context, key, _ string, body []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = body
	return nil
}
func (f *fakeStore) Exists(_ context.Context, key string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.objects[key]
	return ok, nil
}

type fakeModerator struct{ flag string }

func (f fakeModerator) Flagged(_ context.Context, texts []string) (bool, error) {
	for _, t := range texts {
		if f.flag != "" && strings.Contains(t, f.flag) {
			return true, nil
		}
	}
	return false, nil
}

func newSpeakingAPI(t *testing.T) (*SpeakingService, *MockTestRepository, *fakeStore, *Test) {
	t.Helper()
	repo := NewMockTestRepository()
	store := newFakeStore()
	official, _ := repo.CreateTest(context.Background(), &Test{Skill: "speaking", TaskType: "full", ContentData: mustMarshal(t, fullSpeakingContent())})
	examiner := NewExaminerAudio(store, nil, defaultExaminerVoice)
	return NewSpeakingService(repo, store, examiner, nil, fakeModerator{flag: "FORBIDDEN"}), repo, store, official
}

func customPart2JSON() json.RawMessage {
	return json.RawMessage(`{"topic":"Describe a place you visited.","bullets":["where it was","when you went","what you did"],"explain":"and explain why you liked it."}`)
}

func TestUploadSlots(t *testing.T) {
	svc, _, _, _ := newSpeakingAPI(t)
	slots, err := svc.UploadSlots(42, 3, "webm")
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 3 || slots[0].Key == slots[1].Key {
		t.Fatalf("slots = %+v", slots)
	}
	for _, s := range slots {
		if !strings.HasPrefix(s.Key, "speaking/42/") || !strings.HasSuffix(s.Key, ".webm") {
			t.Errorf("key %q is not under the user's prefix", s.Key)
		}
	}
	if _, err := svc.UploadSlots(42, 3, "exe"); !errors.Is(err, ErrInvalidSpeakingContent) {
		t.Errorf("ext exe: err = %v", err)
	}
	if _, err := svc.UploadSlots(42, maxUploadSlots+1, "webm"); !errors.Is(err, ErrInvalidSpeakingContent) {
		t.Errorf("too many slots: err = %v", err)
	}
}

func TestCreateCustom_MixesBankAndOwnParts(t *testing.T) {
	svc, repo, _, official := newSpeakingAPI(t)
	ctx := context.Background()

	res, err := svc.CreateCustom(ctx, 7, ComposeSpeakingRequest{
		Title: "My travel test",
		Part1: &PartSource{BankTestID: official.ID},
		Part2: &PartSource{Custom: customPart2JSON()},
		Part3: &PartSource{BankTestID: official.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	saved, _ := repo.GetTestByID(ctx, res.Test.ID)
	if saved.OwnerID != 7 || saved.TaskType != SpeakingModeFull || saved.XPGain != 0 {
		t.Errorf("saved test = owner %d, task %q, xp %d", saved.OwnerID, saved.TaskType, saved.XPGain)
	}
	var c SpeakingContent
	_ = json.Unmarshal(saved.ContentData, &c)
	if c.Part2.Topic != "Describe a place you visited." || c.Part2.ID != "p2" {
		t.Errorf("part2 = %+v", c.Part2)
	}
	// The bank Part 3 keeps its own Part 2's topic as its theme.
	if c.Part3.Theme != "a book you enjoyed reading" {
		t.Errorf("part3 theme = %q", c.Part3.Theme)
	}
	if !res.Test.IsCustom {
		t.Error("response doesn't mark the test as custom")
	}
}

func TestCreateCustom_Rejections(t *testing.T) {
	svc, _, _, official := newSpeakingAPI(t)
	ctx := context.Background()

	cases := map[string]struct {
		req  ComposeSpeakingRequest
		want error
	}{
		"two parts":    {ComposeSpeakingRequest{Part1: &PartSource{BankTestID: official.ID}, Part2: &PartSource{Custom: customPart2JSON()}}, ErrInvalidSpeakingContent},
		"both sources": {ComposeSpeakingRequest{Part2: &PartSource{BankTestID: official.ID, Custom: customPart2JSON()}}, ErrInvalidSpeakingContent},
		"missing bank": {ComposeSpeakingRequest{Part2: &PartSource{BankTestID: 999}}, ErrInvalidSpeakingContent},
		"flagged":      {ComposeSpeakingRequest{Title: "FORBIDDEN words", Part2: &PartSource{Custom: customPart2JSON()}}, ErrContentFlagged},
	}
	for name, c := range cases {
		if _, err := svc.CreateCustom(ctx, 7, c.req); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}

	// Another user's custom test is not a bank.
	mine, _ := svc.CreateCustom(ctx, 7, ComposeSpeakingRequest{Part2: &PartSource{Custom: customPart2JSON()}})
	if _, err := svc.CreateCustom(ctx, 8, ComposeSpeakingRequest{Part2: &PartSource{BankTestID: mine.Test.ID}}); !errors.Is(err, ErrInvalidSpeakingContent) {
		t.Errorf("copying from someone's custom test: err = %v", err)
	}
}

func TestCustomTest_OwnershipAndAttempts(t *testing.T) {
	svc, repo, _, _ := newSpeakingAPI(t)
	ctx := context.Background()
	req := ComposeSpeakingRequest{Part2: &PartSource{Custom: customPart2JSON()}}
	res, _ := svc.CreateCustom(ctx, 7, req)
	id := res.Test.ID

	if _, err := svc.UpdateCustom(ctx, 8, id, req); !errors.Is(err, ErrTestNotFound) {
		t.Errorf("another user updated it: err = %v", err)
	}
	if err := svc.DeleteCustom(ctx, 8, id); !errors.Is(err, ErrTestNotFound) {
		t.Errorf("another user deleted it: err = %v", err)
	}
	if _, err := svc.Script(ctx, 8, id); !errors.Is(err, ErrTestNotFound) {
		t.Errorf("another user got its script: err = %v", err)
	}
	if list, _ := svc.ListCustom(ctx, 8, 1); len(list.Data) != 0 {
		t.Errorf("another user lists it: %+v", list.Data)
	}

	_, _ = repo.CreateSubmission(ctx, &Submission{UserID: 7, TestID: id, Payload: []byte(`{}`)})
	if _, err := svc.UpdateCustom(ctx, 7, id, req); !errors.Is(err, ErrTestHasAttempts) {
		t.Errorf("edited after an attempt: err = %v", err)
	}
	if err := svc.DeleteCustom(ctx, 7, id); !errors.Is(err, ErrTestHasAttempts) {
		t.Errorf("deleted after an attempt: err = %v", err)
	}
}

func TestScript_UsesRecordedExaminerLinesWhenPresent(t *testing.T) {
	svc, _, store, official := newSpeakingAPI(t)
	ctx := context.Background()

	var c SpeakingContent
	_ = json.Unmarshal(official.ContentData, &c)
	lines := buildSpeakingScript(c, defaultExaminerVoice)
	_ = store.Put(ctx, lines[0].AudioKey, "audio/mpeg", []byte("mp3"))

	script, err := svc.Script(ctx, 1, official.ID)
	if err != nil {
		t.Fatal(err)
	}
	if script.Mode != SpeakingModeFull {
		t.Errorf("mode = %q", script.Mode)
	}
	if script.Lines[0].AudioURL == "" {
		t.Error("recorded line has no audio_url")
	}
	if script.Lines[1].AudioURL != "" {
		t.Error("unrecorded line has an audio_url; the runner should fall back to browser speech")
	}
}

func TestRecordings_OwnerOnly(t *testing.T) {
	svc, repo, _, official := newSpeakingAPI(t)
	ctx := context.Background()
	payload := SpeakingPayload{Answers: []SpeakingAnswer{{QuestionID: "p2", AudioKey: "speaking/7/a.webm", DurationSec: 90}}}
	sub, _ := repo.CreateSubmission(ctx, &Submission{UserID: 7, TestID: official.ID, Payload: mustMarshal(t, payload)})

	links, err := svc.Recordings(ctx, 7, sub.ID)
	if err != nil || links["p2"] == "" {
		t.Errorf("owner's recordings = %v, %v", links, err)
	}
	if _, err := svc.Recordings(ctx, 8, sub.ID); !errors.Is(err, ErrSubmissionNotFound) {
		t.Errorf("another user: err = %v", err)
	}
}
