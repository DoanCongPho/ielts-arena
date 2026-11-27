package music

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type memRepo struct {
	mu     sync.Mutex
	tracks []Track
	next   uint64
	failOn string
}

func (m *memRepo) List(_ context.Context, userID uint64) ([]Track, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Track{}
	for _, t := range m.tracks {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *memRepo) Count(ctx context.Context, userID uint64) (int, error) {
	l, _ := m.List(ctx, userID)
	return len(l), nil
}

func (m *memRepo) Create(_ context.Context, t *Track) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failOn == "create" {
		return errors.New("db down")
	}
	m.next++
	t.ID = m.next
	m.tracks = append(m.tracks, *t)
	return nil
}

func (m *memRepo) Get(_ context.Context, userID, id uint64) (*Track, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tracks {
		if t.ID == id && t.UserID == userID {
			c := t
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *memRepo) Delete(_ context.Context, userID, id uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.tracks {
		if t.ID == id && t.UserID == userID {
			m.tracks = append(m.tracks[:i], m.tracks[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

type memStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newMemStore() *memStore { return &memStore{objects: map[string][]byte{}} }

func (s *memStore) UploadURL(key string, _ time.Duration) (string, error) { return "put:" + key, nil }
func (s *memStore) DownloadURL(key string, _ time.Duration) (string, error) {
	return "get:" + key, nil
}
func (s *memStore) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.objects[key], nil
}
func (s *memStore) Put(_ context.Context, key, _ string, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = body
	return nil
}
func (s *memStore) Exists(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objects[key]
	return ok, nil
}
func (s *memStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}

// An MP3 starting with an ID3 tag, as most do.
var mp3 = append([]byte("ID3\x04\x00\x00\x00\x00\x00\x00"), make([]byte, 2048)...)

// upload goes through the whole flow: ask for a slot, PUT the bytes (here
// straight into the store), confirm.
func upload(svc Service, store *memStore, userID uint64, filename, title string, body []byte) (*TrackView, error) {
	ctx := context.Background()
	slot, err := svc.RequestUpload(ctx, userID, filename, len(body))
	if err != nil {
		return nil, err
	}
	if err := store.Put(ctx, slot.Key, slot.ContentType, body); err != nil {
		return nil, err
	}
	return svc.Confirm(ctx, userID, slot.Key, filename, title)
}

func TestUploadStoresUnderTheUsersPrefixAndLists(t *testing.T) {
	repo, store := &memRepo{}, newMemStore()
	svc := NewService(repo, store)

	v, err := upload(svc, store, 7, "Last Christmas.mp3", "", mp3)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if v.Title != "Last Christmas" {
		t.Errorf("title = %q, want the file name without extension", v.Title)
	}
	if !strings.HasPrefix(v.URL, "get:music/7/") || !strings.HasSuffix(v.URL, ".mp3") {
		t.Errorf("url = %q, want a link to music/7/<random>.mp3", v.URL)
	}

	lib, err := svc.Library(context.Background(), 7)
	if err != nil || len(lib.Tracks) != 1 {
		t.Fatalf("library = %+v, %v; want one track", lib, err)
	}
	if other, _ := svc.Library(context.Background(), 8); len(other.Tracks) != 0 {
		t.Error("another learner must not see this track")
	}
}

func TestRequestUploadChecksBeforeAnythingIsSent(t *testing.T) {
	svc := NewService(&memRepo{}, newMemStore())
	ctx := context.Background()
	if _, err := svc.RequestUpload(ctx, 1, "song.exe", 100); !errors.Is(err, ErrNotAudio) {
		t.Errorf("exe: err = %v, want ErrNotAudio", err)
	}
	if _, err := svc.RequestUpload(ctx, 1, "song.mp3", MaxTrackBytes+1); !errors.Is(err, ErrTooLarge) {
		t.Errorf("too big: err = %v, want ErrTooLarge", err)
	}
	if _, err := svc.RequestUpload(ctx, 1, "song.mp3", 0); !errors.Is(err, ErrEmptyUpload) {
		t.Errorf("empty: err = %v, want ErrEmptyUpload", err)
	}
}

func TestConfirmRefusesAndDeletesWhatIsNotAudio(t *testing.T) {
	store := newMemStore()
	svc := NewService(&memRepo{}, store)
	page := []byte("<!doctype html><html><body>hi</body></html>")
	if _, err := upload(svc, store, 1, "page.mp3", "", page); !errors.Is(err, ErrNotAudio) {
		t.Fatalf("err = %v, want ErrNotAudio", err)
	}
	if len(store.objects) != 0 {
		t.Errorf("objects left = %d, want the refused upload deleted", len(store.objects))
	}
}

func TestConfirmOnlyYourOwnKeys(t *testing.T) {
	store := newMemStore()
	svc := NewService(&memRepo{}, store)
	ctx := context.Background()
	slot, err := svc.RequestUpload(ctx, 1, "a.mp3", len(mp3))
	if err != nil {
		t.Fatal(err)
	}
	_ = store.Put(ctx, slot.Key, slot.ContentType, mp3)
	for _, key := range []string{slot.Key, "music/1/../2/x.mp3", "examiner/x.mp3"} {
		if _, err := svc.Confirm(ctx, 2, key, "a.mp3", ""); !errors.Is(err, ErrNotFound) {
			t.Errorf("user 2 confirming %q: err = %v, want ErrNotFound", key, err)
		}
	}
	if _, ok := store.objects[slot.Key]; !ok {
		t.Error("a refused claim must not delete the owner's object")
	}
}

func TestUploadStopsAtTheTrackLimit(t *testing.T) {
	store := newMemStore()
	svc := NewService(&memRepo{}, store)
	for i := 0; i < MaxTracks; i++ {
		if _, err := upload(svc, store, 1, "a.mp3", "", mp3); err != nil {
			t.Fatalf("upload %d: %v", i, err)
		}
	}
	if _, err := upload(svc, store, 1, "a.mp3", "", mp3); !errors.Is(err, ErrTooMany) {
		t.Fatalf("err = %v, want ErrTooMany", err)
	}
}

func TestFailedInsertLeavesNoObject(t *testing.T) {
	repo, store := &memRepo{failOn: "create"}, newMemStore()
	if _, err := upload(NewService(repo, store), store, 1, "a.mp3", "", mp3); err == nil {
		t.Fatal("expected the insert error")
	}
	if len(store.objects) != 0 {
		t.Errorf("objects left = %d, want 0", len(store.objects))
	}
}

func TestDeleteOnlyYourOwnTrack(t *testing.T) {
	repo, store := &memRepo{}, newMemStore()
	svc := NewService(repo, store)
	ctx := context.Background()
	v, err := upload(svc, store, 1, "a.mp3", "Mine", mp3)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(ctx, 2, v.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("someone else's delete: err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, 1, v.ID); err != nil {
		t.Fatalf("own delete: %v", err)
	}
	if len(store.objects) != 0 || len(repo.tracks) != 0 {
		t.Errorf("left %d objects, %d rows; want none", len(store.objects), len(repo.tracks))
	}
}

func TestTitleIsTrimmedAndCut(t *testing.T) {
	long := strings.Repeat("á", 200)
	if got := cleanTitle("  "+long+"  ", "x.mp3"); len([]rune(got)) != maxTitleRunes {
		t.Errorf("title runes = %d, want %d", len([]rune(got)), maxTitleRunes)
	}
	if got := cleanTitle("", ".mp3"); got != "Bài hát của tôi" {
		t.Errorf("title for a nameless file = %q", got)
	}
}
