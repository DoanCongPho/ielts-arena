package grading

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/feature/speaking/speakingtest"
	"github/DoanCongPho/game-arena/internal/platform/llm"
)

// answerAll records the same two-sentence answer for every question of
// content, as fakeASR "transcribes" a recording's bytes as its text.
func answerAll(content ielts_test.SpeakingContent) (ielts_test.SpeakingPayload, fakeAudio) {
	sentence := "I think it is a really good question and I would say that it depends on the situation because people differ"
	audio := fakeAudio{}
	var payload ielts_test.SpeakingPayload
	for _, q := range content.Questions() {
		key := "speaking/1/" + q.ID + ".webm"
		audio[key] = []byte(sentence + " " + sentence)
		payload.Answers = append(payload.Answers, ielts_test.SpeakingAnswer{QuestionID: q.ID, AudioKey: key, DurationSec: 20})
	}
	return payload, audio
}

func TestGrade_AgainstDescriptorsWithCaps(t *testing.T) {
	content := speakingtest.FullContent()
	payload, audio := answerAll(content)
	judge := &fakeJudge{band: 7}
	g := New(audio, fakeASR{}, &fakePron{}, judge, Config{JudgeModel: "judge", ScorePronunciation: true})

	d, overall, err := g.Grade(context.Background(), content, payload, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(judge.prompts) != 4 {
		t.Errorf("judge called %d times, want one per criterion", len(judge.prompts))
	}
	for _, p := range judge.prompts {
		if !strings.Contains(p, "=== Part 3 ===") || !strings.Contains(p, "Band 7:") {
			t.Error("a judge prompt is missing the whole-test transcript or the descriptors")
		}
	}
	if d.Indicative || d.Mode != ielts_test.SpeakingModeFull {
		t.Errorf("mode = %q, indicative = %v", d.Mode, d.Indicative)
	}
	if d.Pronunciation == nil || d.Pronunciation.Estimated {
		t.Errorf("pronunciation = %+v, want measured", d.Pronunciation)
	}
	// The fake long turn is ~7 s, so the guardrail lowers FC to 5 and the
	// overall is (5+7+7+7)/4 = 6.5.
	if fc := d.Criteria[CriterionFC]; fc.Score != 5 || fc.JudgedBand != 7 || fc.Cap == nil {
		t.Errorf("FC = %+v, want capped from 7 to 5", fc)
	}
	if overall != 6.5 {
		t.Errorf("overall = %v, want 6.5", overall)
	}
}

func TestGrade_PronunciationOutage(t *testing.T) {
	content := speakingtest.FullContent()
	payload, audio := answerAll(content)
	g := New(audio, fakeASR{}, &fakePron{err: errors.New("pronunciation service unreachable")}, &fakeJudge{band: 6}, Config{ScorePronunciation: true})

	// Before the last attempt the job fails, so it is retried.
	if _, _, err := g.Grade(context.Background(), content, payload, false); !errors.Is(err, errPronunciationUnavailable) {
		t.Fatalf("err = %v, want errPronunciationUnavailable", err)
	}
	// On the last attempt pronunciation is estimated instead.
	d, _, err := g.Grade(context.Background(), content, payload, true)
	if err != nil {
		t.Fatalf("last attempt: %v", err)
	}
	if d.Pronunciation == nil || !d.Pronunciation.Estimated {
		t.Errorf("pronunciation = %+v, want estimated", d.Pronunciation)
	}
}

func TestGrade_PartPracticeIsIndicative(t *testing.T) {
	content := speakingtest.FullContent()
	content.Part1, content.Part3 = nil, nil
	payload, audio := answerAll(content)
	judge := &fakeJudge{band: 6}
	g := New(audio, fakeASR{}, &fakePron{}, judge, Config{ScorePronunciation: true})

	d, _, err := g.Grade(context.Background(), content, payload, false)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Indicative || d.Mode != ielts_test.SpeakingModePart2 {
		t.Errorf("mode = %q, indicative = %v; want part2, true", d.Mode, d.Indicative)
	}
	if !strings.Contains(judge.prompts[0], "practised on its own") {
		t.Error("the judge wasn't told this is part practice")
	}
}

func TestGradeSpeaking_StoresTheDetailsAsJSON(t *testing.T) {
	content := speakingtest.FullContent()
	payload, audio := answerAll(content)
	g := New(audio, fakeASR{}, &fakePron{}, &fakeJudge{band: 7}, Config{ScorePronunciation: true})

	overall, raw, err := g.GradeSpeaking(context.Background(), content, payload, false)
	if err != nil {
		t.Fatal(err)
	}
	var d Details
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("stored details are not Details JSON: %v", err)
	}
	if overall != 6.5 || d.Criteria[CriterionLR].Score != 7 {
		t.Errorf("overall %v, LR %v", overall, d.Criteria[CriterionLR].Score)
	}
}

func TestGrade_WithoutPronunciationScoring(t *testing.T) {
	content := speakingtest.FullContent()
	payload, audio := answerAll(content)
	judge := &fakeJudge{band: 7}
	pron := &fakePron{}
	g := New(audio, fakeASR{}, pron, judge, Config{})

	d, overall, err := g.Grade(context.Background(), content, payload, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(judge.prompts) != 3 || pron.calls != 0 {
		t.Errorf("judge called %d times, pronunciation service %d; want 3 and 0", len(judge.prompts), pron.calls)
	}
	if _, ok := d.Criteria[CriterionP]; ok || d.Pronunciation != nil {
		t.Error("pronunciation was rated although scoring it is off")
	}
	// FC is capped to 5 by the short fake long turn: (5+7+7)/3 = 6.33 → 6.5.
	if overall != 6.5 {
		t.Errorf("overall = %v, want 6.5", overall)
	}
}

func TestGrade_Part2RewriteForMrSonsStudents(t *testing.T) {
	content := speakingtest.FullContent()
	payload, audio := answerAll(content)
	judge := &fakeJudge{band: 7}
	g := New(audio, fakeASR{}, nil, judge, Config{})

	d, _, err := g.Grade(context.Background(), content, payload, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(judge.rewrites) != 1 || !strings.Contains(judge.rewrites[0], content.Part2.CueCardText()) {
		t.Fatalf("rewrite prompts = %q, want one with the cue card", judge.rewrites)
	}
	if !strings.Contains(d.Part2Rewrite, "Bài sửa theo 4 tầng") {
		t.Errorf("Part2Rewrite = %q", d.Part2Rewrite)
	}

	// Part 1 practice has no talk to rewrite.
	content = speakingtest.FullContent()
	content.Part2, content.Part3 = nil, nil
	payload, audio = answerAll(content)
	judge = &fakeJudge{band: 7}
	d, _, err = New(audio, fakeASR{}, nil, judge, Config{}).Grade(context.Background(), content, payload, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(judge.rewrites) != 0 || d.Part2Rewrite != "" {
		t.Errorf("Part 1 practice got a Part 2 rewrite: %q", d.Part2Rewrite)
	}
}

func TestGrade_Part2RewriteFailureKeepsTheGrade(t *testing.T) {
	content := speakingtest.FullContent()
	payload, audio := answerAll(content)
	g := New(audio, fakeASR{}, nil, &brokenRewriteJudge{fakeJudge{band: 6}}, Config{})

	d, overall, err := g.Grade(context.Background(), content, payload, false)
	if err != nil {
		t.Fatalf("a failed rewrite failed the grade: %v", err)
	}
	if d.Part2Rewrite != "" || overall == 0 {
		t.Errorf("rewrite %q, overall %v", d.Part2Rewrite, overall)
	}
}

// brokenRewriteJudge judges normally but fails the Part 2 rewrite.
type brokenRewriteJudge struct{ fakeJudge }

func (b *brokenRewriteJudge) Complete(ctx context.Context, system, user, image string, p llm.CompletionParams) (string, error) {
	if system == part2RewriteSystem {
		return "", errors.New("rate limited")
	}
	return b.fakeJudge.Complete(ctx, system, user, image, p)
}
