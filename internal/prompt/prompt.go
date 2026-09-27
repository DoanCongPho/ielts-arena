// Package prompt holds prompts written by teachers, embedded exactly as
// they wrote them so they can keep editing the Markdown files.
package prompt

import _ "embed"

// SpeakingPart2MrSon is Mr Sơn's "Flow 4 tầng" method for correcting a
// Part 2 talk: feedback in Vietnamese and a model answer rebuilt from the
// student's own story, for his students.
//
//go:embed speaking/rewrite_mr.Son_part2.md
var SpeakingPart2MrSon string
