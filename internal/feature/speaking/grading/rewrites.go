package grading

import (
	"context"
	"encoding/json"
	"fmt"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"github/DoanCongPho/game-arena/internal/prompt"
	"log"
	"sort"
	"strings"
	"sync"
)

// PartRewrite is a teacher's review of one part, with a model answer built
// from the student's own words (see internal/prompt/speaking).
type PartRewrite struct {
	Part     int    `json:"part"`
	Title    string `json:"title"`
	Badge    string `json:"badge,omitempty"`
	Note     string `json:"note,omitempty"`
	Markdown string `json:"markdown"`
}

// rewriteOutputFormat follows every teacher prompt: the completer only
// returns JSON.
const rewriteOutputFormat = `

---

## ĐỊNH DẠNG TRẢ VỀ CHO ỨNG DỤNG

Phản hồi được hiển thị trong ứng dụng luyện thi, ngay dưới kết quả chấm. Trả về DUY NHẤT một JSON hợp lệ:
{"markdown": "<toàn bộ phản hồi theo định dạng đầu ra ở trên, viết bằng Markdown, dùng ### cho tiêu đề mục>"}`

// rewriteParams keeps a reasoning model's thinking low: the review follows
// the teacher's fixed format, and thinking was two thirds of its cost.
var rewriteParams = llm.CompletionParams{Temperature: 0.4, MaxTokens: 8000, ReasoningEffort: "low"}

// loadRewrites reads the teachers' prompts. They are embedded, so an error
// is a malformed file; grading goes on without reviews rather than fail.
func loadRewrites() map[int]prompt.SpeakingRewrite {
	r, err := prompt.SpeakingRewrites()
	if err != nil {
		log.Printf("speaking: review prompts: %v — no reviews", err)
	}
	return r
}

// rewriteAll runs each part's review that has a prompt and an answer,
// in parallel. Each call sees only its own part's answers.
func (g *Grader) rewriteAll(ctx context.Context, content ielts_test.SpeakingContent, answers []answerTranscript) []PartRewrite {
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		out []PartRewrite
	)
	for part, p := range g.rewrites {
		input := rewriteInput(part, content, answers)
		if input == "" {
			continue
		}
		wg.Add(1)
		go func(p prompt.SpeakingRewrite) {
			defer wg.Done()
			md := g.rewrite1(ctx, p, input)
			if md == "" {
				return
			}
			mu.Lock()
			out = append(out, PartRewrite{Part: p.Part, Title: p.Title, Badge: p.Badge, Note: p.Note, Markdown: md})
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool { return out[i].Part < out[j].Part })
	return out
}

// rewrite1 runs one part's review. It is an extra, not part of the band,
// so a failure is logged and the grade goes on without it.
func (g *Grader) rewrite1(ctx context.Context, p prompt.SpeakingRewrite, input string) string {
	params := rewriteParams
	params.Model = g.cfg.ReviewModel
	if params.Model == "" {
		params.Model = g.cfg.JudgeModel
	}
	raw, err := g.judge.Complete(ctx, p.System+rewriteOutputFormat, input, "", params)
	if err != nil {
		log.Printf("speaking: part %d review: %v", p.Part, err)
		return ""
	}
	var res struct {
		Markdown string `json:"markdown"`
	}
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		log.Printf("speaking: part %d review: parse: %v", p.Part, err)
		return ""
	}
	return strings.TrimSpace(res.Markdown)
}

// rewriteInput is what one part's review is given: Part 2's cue card and
// talk, or Part 1 and 3's questions with the answers. Empty when the
// candidate didn't answer that part.
func rewriteInput(part int, content ielts_test.SpeakingContent, answers []answerTranscript) string {
	const asr = "nhận dạng giọng nói tự động nên có thể nghe nhầm vài từ"
	if part == 2 {
		if content.Part2 == nil {
			return ""
		}
		for _, a := range answers {
			if a.QuestionID == content.Part2.ID && a.Text != "" {
				return fmt.Sprintf("Đề bài (cue card):\n%s\n\nBản ghi lời nói của học viên (%.0f giây, %s):\n%s",
					content.Part2.CueCardText(), a.Duration, asr, a.Text)
			}
		}
		return ""
	}
	var b strings.Builder
	for _, a := range answers {
		if a.Part != part || a.Text == "" {
			continue
		}
		fmt.Fprintf(&b, "\nGiám khảo: %s\nHọc viên (%.0f giây): %s\n", a.Question, a.Duration, a.Text)
	}
	if b.Len() == 0 {
		return ""
	}
	return fmt.Sprintf("Các câu hỏi Part %d và câu trả lời của học viên (%s):\n%s", part, asr, b.String())
}
