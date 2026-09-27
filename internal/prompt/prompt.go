// Package prompt holds prompts written by teachers, embedded exactly as
// they wrote them so they can keep editing the Markdown files.
package prompt

import (
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
)

// speaking/partN.md is the review prompt for Part N. A part without a file
// gets no review, so adding or removing a file is all it takes.
//
//go:embed speaking
var speakingFiles embed.FS

// SpeakingRewrite is a teacher's prompt for reviewing one part of a
// speaking test and writing a model answer from the student's own words.
type SpeakingRewrite struct {
	Part int
	// Title, Badge and Note head the review on the result page. They come
	// from the file's front matter.
	Title string
	Badge string
	Note  string
	// System is the prompt itself: the file without its front matter.
	System string
}

var partFile = regexp.MustCompile(`^part([1-3])\.md$`)

// SpeakingRewrites returns the review prompts by part.
func SpeakingRewrites() (map[int]SpeakingRewrite, error) {
	return loadSpeakingRewrites(speakingFiles)
}

func loadSpeakingRewrites(fsys fs.FS) (map[int]SpeakingRewrite, error) {
	entries, err := fs.ReadDir(fsys, "speaking")
	if err != nil {
		return nil, err
	}
	out := map[int]SpeakingRewrite{}
	for _, e := range entries {
		m := partFile.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		raw, err := fs.ReadFile(fsys, "speaking/"+e.Name())
		if err != nil {
			return nil, err
		}
		part, _ := strconv.Atoi(m[1])
		r, err := parseRewrite(part, string(raw))
		if err != nil {
			return nil, fmt.Errorf("speaking/%s: %w", e.Name(), err)
		}
		out[part] = r
	}
	return out, nil
}

// parseRewrite splits off the front matter: "key: value" lines between
// two "---" lines at the top of the file.
func parseRewrite(part int, raw string) (SpeakingRewrite, error) {
	r := SpeakingRewrite{Part: part, Title: fmt.Sprintf("Bài mẫu Part %d", part)}
	body := strings.ReplaceAll(raw, "\r\n", "\n")
	if rest, ok := strings.CutPrefix(body, "---\n"); ok {
		header, prompt, found := strings.Cut(rest, "\n---\n")
		if !found {
			return r, fmt.Errorf("front matter has no closing ---")
		}
		for _, line := range strings.Split(header, "\n") {
			key, value, _ := strings.Cut(line, ":")
			value = strings.TrimSpace(value)
			switch strings.TrimSpace(key) {
			case "title":
				r.Title = value
			case "badge":
				r.Badge = value
			case "note":
				r.Note = value
			}
		}
		body = prompt
	}
	r.System = strings.TrimSpace(body)
	if r.System == "" {
		return r, fmt.Errorf("the prompt is empty")
	}
	return r, nil
}
