package grading

import (
	"context"
	"encoding/json"
	"fmt"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"github/DoanCongPho/game-arena/internal/prompt"
	"log"
	"strings"
)

// part2RewriteSystem is Mr Sơn's prompt, plus how the app needs the answer
// back: the completer only returns JSON.
var part2RewriteSystem = prompt.SpeakingPart2MrSon + `

---

## ĐỊNH DẠNG TRẢ VỀ CHO ỨNG DỤNG

Phản hồi được hiển thị trong ứng dụng luyện thi, ngay dưới kết quả chấm. Trả về DUY NHẤT một JSON hợp lệ:
{"markdown": "<toàn bộ phản hồi theo các mục ở phần 5, viết bằng Markdown, dùng ### cho tiêu đề mục>"}
Học viên xem lại trên ứng dụng, nên thay câu mời gửi bản ghi tiếp theo bằng một câu mời thử nói lại bài theo ghi chú ở Mục 7.`

var part2RewriteParams = llm.CompletionParams{Temperature: 0.4, MaxTokens: 4000}

// rewritePart2 runs Mr Sơn's 4-layer review of the Part 2 talk. It is an
// extra for his students, not part of the band, so a failure is logged
// and the grade goes on without it.
func (g *Grader) rewritePart2(ctx context.Context, content ielts_test.SpeakingContent, answers []answerTranscript) string {
	p2 := content.Part2
	if p2 == nil {
		return ""
	}
	var talk *answerTranscript
	for i := range answers {
		if answers[i].QuestionID == p2.ID {
			talk = &answers[i]
		}
	}
	if talk == nil || talk.Text == "" {
		return ""
	}
	user := fmt.Sprintf("Đề bài (cue card):\n%s\n\nBản ghi lời nói của học viên (%.0f giây, nhận dạng giọng nói tự động nên có thể nghe nhầm vài từ):\n%s",
		p2.CueCardText(), talk.Duration, talk.Text)

	params := part2RewriteParams
	params.Model = g.cfg.JudgeModel
	raw, err := g.judge.Complete(ctx, part2RewriteSystem, user, "", params)
	if err != nil {
		log.Printf("speaking: part 2 rewrite: %v", err)
		return ""
	}
	var out struct {
		Markdown string `json:"markdown"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		log.Printf("speaking: part 2 rewrite: parse: %v", err)
		return ""
	}
	return strings.TrimSpace(out.Markdown)
}
