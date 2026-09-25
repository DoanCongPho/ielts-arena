package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
)

// Recording formats browsers produce with MediaRecorder: Chrome and Firefox
// record WebM/Ogg Opus, Safari MP4/AAC. Whisper accepts all of them.
var recordingExts = map[string]bool{"webm": true, "ogg": true, "mp4": true, "m4a": true}

// UploadSlot is where the browser PUTs one recording, and the key the
// submission then names it by.
type UploadSlot struct {
	Key       string `json:"key"`
	UploadURL string `json:"upload_url"`
}

func (s *Service) UploadSlots(userID uint64, count int, ext string) ([]UploadSlot, error) {
	if count < 1 || count > maxUploadSlots {
		return nil, fmt.Errorf("%w: count must be 1-%d", ErrInvalidContent, maxUploadSlots)
	}
	if !recordingExts[ext] {
		return nil, fmt.Errorf("%w: unsupported recording format %q", ErrInvalidContent, ext)
	}
	slots := make([]UploadSlot, count)
	for i := range slots {
		var id [12]byte
		if _, err := rand.Read(id[:]); err != nil {
			return nil, err
		}
		key := ielts_test.SpeakingAudioPrefix(userID) + hex.EncodeToString(id[:]) + "." + ext
		link, err := s.store.UploadURL(key, uploadLinkTTL)
		if err != nil {
			return nil, err
		}
		slots[i] = UploadSlot{Key: key, UploadURL: link}
	}
	return slots, nil
}
