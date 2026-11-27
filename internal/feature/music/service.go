package music

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github/DoanCongPho/game-arena/internal/platform/storage"
)

// audioTypes are the file kinds the player can play, by extension. The
// browser's declared type is unreliable (often empty or octet-stream), so
// the extension decides, and the bytes are sniffed to refuse anything that
// is plainly not audio (a renamed web page or image).
var audioTypes = map[string]string{
	".mp3":  "audio/mpeg",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".opus": "audio/ogg",
	".wav":  "audio/wav",
	".webm": "audio/webm",
	".flac": "audio/flac",
}

// UploadSlot is where the browser PUTs a track's bytes: straight to the
// bucket (or a signed path on this server without one), so large files
// never pass through the Vercel proxy or the API's memory on the way in.
type UploadSlot struct {
	Key         string `json:"key"`
	UploadURL   string `json:"upload_url"`
	ContentType string `json:"content_type"`
}

type Service interface {
	Library(ctx context.Context, userID uint64) (*Library, error)
	// RequestUpload checks a file before it's sent and hands out a slot.
	RequestUpload(ctx context.Context, userID uint64, filename string, size int) (*UploadSlot, error)
	// Confirm adds an uploaded object to the learner's library once it has
	// checked the bytes really are audio and within the limits; anything
	// refused is deleted from the store.
	Confirm(ctx context.Context, userID uint64, key, filename, title string) (*TrackView, error)
	Delete(ctx context.Context, userID, id uint64) error
}

type service struct {
	repo  Repository
	store storage.ObjectStore
	now   func() time.Time
}

func NewService(repo Repository, store storage.ObjectStore) Service {
	return &service{repo: repo, store: store, now: time.Now}
}

func (s *service) view(t Track) (TrackView, error) {
	link, err := s.store.DownloadURL(t.ObjectKey, linkTTL)
	if err != nil {
		return TrackView{}, fmt.Errorf("track link: %w", err)
	}
	return TrackView{ID: t.ID, Title: t.Title, URL: link, SizeBytes: t.SizeBytes, CreatedAt: t.CreatedAt}, nil
}

func (s *service) Library(ctx context.Context, userID uint64) (*Library, error) {
	tracks, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	lib := &Library{Tracks: make([]TrackView, 0, len(tracks)), MaxBytes: MaxTrackBytes, MaxCount: MaxTracks}
	for _, t := range tracks {
		v, err := s.view(t)
		if err != nil {
			return nil, err
		}
		lib.Tracks = append(lib.Tracks, v)
	}
	return lib, nil
}

func (s *service) checkRoom(ctx context.Context, userID uint64) error {
	n, err := s.repo.Count(ctx, userID)
	if err != nil {
		return err
	}
	if n >= MaxTracks {
		return ErrTooMany
	}
	return nil
}

func (s *service) RequestUpload(ctx context.Context, userID uint64, filename string, size int) (*UploadSlot, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	contentType, ok := audioTypes[ext]
	if !ok {
		return nil, ErrNotAudio
	}
	if size <= 0 {
		return nil, ErrEmptyUpload
	}
	if size > MaxTrackBytes {
		return nil, ErrTooLarge
	}
	if err := s.checkRoom(ctx, userID); err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s%s%s", userPrefix(userID), randomHex(12), ext)
	link, err := s.store.UploadURL(key, uploadTTL)
	if err != nil {
		return nil, fmt.Errorf("upload link: %w", err)
	}
	return &UploadSlot{Key: key, UploadURL: link, ContentType: contentType}, nil
}

func (s *service) Confirm(ctx context.Context, userID uint64, key, filename, title string) (*TrackView, error) {
	// Only keys handed out to this learner, so nobody can claim (or make
	// us delete) someone else's object.
	if !strings.HasPrefix(key, userPrefix(userID)) || strings.Contains(key[len(userPrefix(userID)):], "/") {
		return nil, ErrNotFound
	}
	ext := strings.ToLower(filepath.Ext(key))
	contentType, ok := audioTypes[ext]
	if !ok {
		return nil, ErrNotFound
	}

	body, err := s.store.Get(ctx, key)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}
	refuse := func(reason error) (*TrackView, error) {
		if derr := s.store.Delete(ctx, key); derr != nil {
			log.Printf("music: delete refused upload %s: %v", key, derr)
		}
		return nil, reason
	}
	switch {
	case len(body) == 0:
		return refuse(ErrEmptyUpload)
	case len(body) > MaxTrackBytes:
		return refuse(ErrTooLarge)
	case !looksLikeAudio(body):
		return refuse(ErrNotAudio)
	}
	if err := s.checkRoom(ctx, userID); err != nil {
		return refuse(err)
	}

	t := &Track{
		UserID:      userID,
		Title:       cleanTitle(title, filename),
		ObjectKey:   key,
		ContentType: contentType,
		SizeBytes:   len(body),
		CreatedAt:   s.now(),
	}
	if err := s.repo.Create(ctx, t); err != nil {
		_, _ = refuse(nil)
		return nil, err
	}
	v, err := s.view(*t)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func userPrefix(userID uint64) string {
	return fmt.Sprintf("music/%d/", userID)
}

func (s *service) Delete(ctx context.Context, userID, id uint64) error {
	t, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return err
	}
	// The row is gone, so the track is gone for the learner; a leftover
	// object is only wasted space, not worth failing the request over.
	if err := s.store.Delete(ctx, t.ObjectKey); err != nil {
		log.Printf("music: delete object %s: %v", t.ObjectKey, err)
	}
	return nil
}

// looksLikeAudio refuses bytes that sniff as text, an image or a document.
// Audio formats sniff as audio/*, video/* (MP4/WebM containers) or, for an
// MP3 without an ID3 tag, application/octet-stream.
func looksLikeAudio(body []byte) bool {
	sniffed := http.DetectContentType(body)
	switch {
	case strings.HasPrefix(sniffed, "audio/"), strings.HasPrefix(sniffed, "video/"),
		strings.HasPrefix(sniffed, "application/octet-stream"), strings.HasPrefix(sniffed, "application/ogg"):
		return true
	default:
		return false
	}
}

// cleanTitle is the given title, or the file's name without its extension,
// trimmed and cut to maxTitleRunes.
func cleanTitle(title, filename string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		base := filepath.Base(filename)
		t = strings.TrimSpace(strings.TrimSuffix(base, filepath.Ext(base)))
	}
	if t == "" || t == "." {
		t = "Bài hát của tôi"
	}
	if utf8.RuneCountInString(t) > maxTitleRunes {
		t = string([]rune(t)[:maxTitleRunes])
	}
	return t
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand doesn't fail on supported platforms
	}
	return hex.EncodeToString(b)
}
