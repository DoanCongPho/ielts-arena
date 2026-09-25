package ielts_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"
)

// Calibration runs examiner-rated recordings through the real grading
// pipeline and reports how often it agrees with the examiners. Bands should
// only be presented as more than estimates once agreement meets the targets
// below; the guardrail thresholds in speaking_scoring.go are what to tune.
const (
	calibrationTargetOverall   = 0.80 // overall within ±0.5 of the examiner
	calibrationTargetCriterion = 0.70 // each criterion exactly right
)

// CalibrationSample is one examiner-rated performance, e.g. a published
// British Council, IDP or Cambridge sample with examiner commentary.
type CalibrationSample struct {
	ID     string `json:"id"`
	Source string `json:"source"` // where the recording and its bands come from
	// Bands holds the examiner's "overall" band and any criterion bands
	// published, keyed by criterion name.
	Bands   map[string]float64  `json:"bands"`
	Answers []CalibrationAnswer `json:"answers"`
}

type CalibrationAnswer struct {
	Part     int    `json:"part"`
	Question string `json:"question"`
	// Audio is the recording's path, relative to the manifest.
	Audio string `json:"audio"`
}

// fileAudio reads recordings from disk: keys are file paths.
type fileAudio struct{}

func (fileAudio) Get(_ context.Context, key string) ([]byte, error) { return os.ReadFile(key) }

// sampleContent turns a sample's answers into test content and a payload
// whose audio keys are file paths.
func sampleContent(s CalibrationSample, dir string) (SpeakingContent, SpeakingPayload) {
	var c SpeakingContent
	for _, a := range s.Answers {
		switch a.Part {
		case 1:
			if c.Part1 == nil {
				c.Part1 = &SpeakingPart1{Topics: []SpeakingTopic{{Topic: "Interview"}}}
			}
			c.Part1.Topics[0].Questions = append(c.Part1.Topics[0].Questions, SpeakingQuestion{Text: a.Question})
		case 2:
			if c.Part2 == nil {
				c.Part2 = &SpeakingPart2{Topic: a.Question}
			} else {
				c.Part2.FollowUps = append(c.Part2.FollowUps, SpeakingQuestion{Text: a.Question})
			}
		case 3:
			if c.Part3 == nil {
				c.Part3 = &SpeakingPart3{}
			}
			c.Part3.Questions = append(c.Part3.Questions, SpeakingQuestion{Text: a.Question})
		}
	}
	normalizeSpeakingContent(&c)

	// Questions come back in exam order; pair them with the answers in
	// the same order.
	byPart := map[int][]CalibrationAnswer{}
	for _, a := range s.Answers {
		byPart[a.Part] = append(byPart[a.Part], a)
	}
	var p SpeakingPayload
	for _, q := range c.questions() {
		a := byPart[q.Part][0]
		byPart[q.Part] = byPart[q.Part][1:]
		p.Answers = append(p.Answers, SpeakingAnswer{QuestionID: q.ID, AudioKey: filepath.Join(dir, a.Audio)})
	}
	return c, p
}

// RunCalibrateCmd implements `api calibrate-speaking MANIFEST.json`.
func RunCalibrateCmd(g *SpeakingGrader, args []string, out io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(out, "usage: api calibrate-speaking MANIFEST.json")
		return 2
	}
	raw, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(out, err)
		return 1
	}
	var samples []CalibrationSample
	if err := json.Unmarshal(raw, &samples); err != nil {
		fmt.Fprintf(out, "manifest: %v\n", err)
		return 1
	}
	g.audio = fileAudio{}
	dir := filepath.Dir(args[0])

	var (
		overallN, overallHit int
		critN                = map[string]int{}
		critHit              = map[string]int{}
	)
	fmt.Fprintf(out, "%-24s %8s %8s   %s\n", "sample", "examiner", "arena", "criteria (examiner/arena)")
	for _, s := range samples {
		content, payload := sampleContent(s, dir)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		details, overall, err := g.Grade(ctx, content, payload, true)
		cancel()
		if err != nil {
			fmt.Fprintf(out, "%-24s error: %v\n", s.ID, err)
			continue
		}
		// SPEAKING_CALIBRATE_DUMP=DIR saves each sample's full result, to
		// see why a band differs from the examiner's.
		if dump := os.Getenv("SPEAKING_CALIBRATE_DUMP"); dump != "" {
			if b, err := json.MarshalIndent(details, "", " "); err == nil {
				_ = os.MkdirAll(dump, 0o755)
				_ = os.WriteFile(filepath.Join(dump, s.ID+".json"), b, 0o644)
			}
		}
		line := ""
		for _, c := range speakingCriteria {
			got := details.Criteria[c].Score
			want, ok := s.Bands[c]
			if !ok {
				// No examiner band to compare with: still show ours.
				line += fmt.Sprintf("%s -/%.0f  ", initials(c), got)
				continue
			}
			critN[c]++
			if got == want {
				critHit[c]++
			}
			line += fmt.Sprintf("%s %.0f/%.0f  ", initials(c), want, got)
		}
		if p := details.Pronunciation; p != nil {
			line += fmt.Sprintf("| pron: intelligible %.0f%%, pitch %.1f st, estimated=%v", 100*p.Intelligibility, p.Prosody.PitchStdST, p.Estimated)
		}
		if want, ok := s.Bands["overall"]; ok {
			overallN++
			if math.Abs(overall-want) <= 0.5 {
				overallHit++
			}
			fmt.Fprintf(out, "%-24s %8.1f %8.1f   %s\n", s.ID, want, overall, line)
		} else {
			fmt.Fprintf(out, "%-24s %8s %8.1f   %s\n", s.ID, "-", overall, line)
		}
	}

	fmt.Fprintln(out)
	pass := true
	if overallN > 0 {
		rate := float64(overallHit) / float64(overallN)
		pass = pass && rate >= calibrationTargetOverall
		fmt.Fprintf(out, "overall within ±0.5: %d/%d (%.0f%%, target %.0f%%)\n", overallHit, overallN, 100*rate, 100*calibrationTargetOverall)
	}
	for _, c := range speakingCriteria {
		if critN[c] == 0 {
			continue
		}
		rate := float64(critHit[c]) / float64(critN[c])
		pass = pass && rate >= calibrationTargetCriterion
		fmt.Fprintf(out, "%-32s exact: %d/%d (%.0f%%, target %.0f%%)\n", c, critHit[c], critN[c], 100*rate, 100*calibrationTargetCriterion)
	}
	if !pass {
		fmt.Fprintln(out, "\nBelow target: tune the thresholds in speaking_scoring.go before showing bands as more than estimates.")
		return 1
	}
	return 0
}

func initials(criterion string) string {
	switch criterion {
	case CriterionFC:
		return "FC"
	case CriterionLR:
		return "LR"
	case CriterionGRA:
		return "GRA"
	}
	return "P"
}
