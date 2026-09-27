package prompt

import (
	"testing"
	"testing/fstest"
)

func TestSpeakingRewrites_OnePerPartFile(t *testing.T) {
	got, err := loadSpeakingRewrites(fstest.MapFS{
		"speaking/part1.md":  {Data: []byte("---\ntitle: Part 1 mẫu\nbadge: Lớp A\n---\nYou review Part 1.\n")},
		"speaking/part3.md":  {Data: []byte("No front matter.")},
		"speaking/README.md": {Data: []byte("not a prompt")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got parts %v, want 1 and 3", got)
	}
	if p1 := got[1]; p1.Title != "Part 1 mẫu" || p1.Badge != "Lớp A" || p1.System != "You review Part 1." {
		t.Errorf("part 1 = %+v", p1)
	}
	if p3 := got[3]; p3.Title != "Bài mẫu Part 3" || p3.System != "No front matter." {
		t.Errorf("part 3 = %+v", p3)
	}
}

func TestSpeakingRewrites_Embedded(t *testing.T) {
	got, err := SpeakingRewrites()
	if err != nil {
		t.Fatal(err)
	}
	p2, ok := got[2]
	if !ok || p2.Badge == "" || p2.System[:1] != "#" {
		t.Errorf("part 2 = %+v, want Mr Sơn's prompt with its badge", p2.Title)
	}
}

func TestSpeakingRewrites_UnclosedFrontMatter(t *testing.T) {
	_, err := loadSpeakingRewrites(fstest.MapFS{"speaking/part2.md": {Data: []byte("---\ntitle: x\nprompt")}})
	if err == nil {
		t.Error("an unclosed front matter was accepted")
	}
}
