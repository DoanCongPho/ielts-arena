// Package music keeps the tracks a learner uploads for the music player.
// A track is private to its uploader: it's listed and played only for
// them. The audio sits in the media object store (bucket in production)
// under music/<user_id>/, and a user_tracks row points at it.
package music

import (
	"errors"
	"time"
)

const (
	// MaxTrackBytes caps one upload. A 4-minute 256 kbps MP3 is ~8 MB.
	MaxTrackBytes = 15 << 20
	// MaxTracks caps how many tracks one learner keeps.
	MaxTracks = 20
	// maxTitleRunes caps a track's title.
	maxTitleRunes = 120
	// linkTTL is how long a track's play link lasts; the player asks for
	// fresh ones each time it loads the list.
	linkTTL = 6 * time.Hour
	// uploadTTL is how long an upload slot accepts the file.
	uploadTTL = 15 * time.Minute
)

var (
	ErrNotFound    = errors.New("track not found")
	ErrTooLarge    = errors.New("track too large")
	ErrTooMany     = errors.New("too many tracks")
	ErrNotAudio    = errors.New("not an audio file")
	ErrEmptyUpload = errors.New("empty upload")
)

type Track struct {
	ID          uint64
	UserID      uint64
	Title       string
	ObjectKey   string
	ContentType string
	SizeBytes   int
	CreatedAt   time.Time
}

// TrackView is a track as its owner's player sees it.
type TrackView struct {
	ID        uint64    `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	SizeBytes int       `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

type Library struct {
	Tracks   []TrackView `json:"tracks"`
	MaxBytes int         `json:"max_bytes"`
	MaxCount int         `json:"max_count"`
}
