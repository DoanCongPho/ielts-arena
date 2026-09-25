package examiner

import (
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/feature/speaking/speakingtest"
	"testing"
)

func TestGolden_SpeakingScript(t *testing.T) {
	type line struct {
		Kind, Text, QuestionID, AudioKey string
		Part, Seconds                    int
	}
	flatten := func(ls []Line) []line {
		out := make([]line, len(ls))
		for i, l := range ls {
			out[i] = line{l.Kind, l.Text, l.QuestionID, l.AudioKey, l.Part, l.Seconds}
		}
		return out
	}
	full := speakingtest.FullContent()
	part2 := speakingtest.FullContent()
	part2.Part1, part2.Part3 = nil, nil
	part3 := ielts_test.SpeakingContent{Part3: &ielts_test.SpeakingPart3{Theme: "Reading habits", Questions: full.Part3.Questions}}
	scripts := map[string][]line{
		"full":  flatten(Build(full, defaultVoice)),
		"part2": flatten(Build(part2, defaultVoice)),
		"part3": flatten(Build(part3, NewVoice("tts-x", "alloy"))),
	}
	speakingtest.CheckGolden(t, "script.json", speakingtest.IndentJSON(t, scripts))
}
