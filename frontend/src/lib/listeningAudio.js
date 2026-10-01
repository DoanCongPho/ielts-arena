// Timing helpers for a listening test's recording: sections either share
// one file (content.audio_url, each a time range in it) or each have their
// own (section.audio_url, timed within that file).

// playlist lists what to play, in order: each section's own recording, or
// the test's one shared recording.
export function playlist(content) {
  const sections = content?.sections || [];
  if (sections.length > 0 && sections.every((s) => s.audio_url)) {
    return sections.map((s) => ({ src: s.audio_url, length: sectionLength(s) }));
  }
  return content?.audio_url ? [{ src: content.audio_url, length: totalSeconds(content) }] : [];
}

function sectionLength(s) {
  return Math.max(0, (s.section_end_time || 0) - (s.section_start_time || 0));
}

// totalSeconds is how long the whole recording runs.
export function totalSeconds(content) {
  return (content?.sections || []).reduce((sum, s) => sum + sectionLength(s), 0);
}

// sectionAt finds which section a position in a shared recording is in.
export function sectionAt(content, seconds) {
  const sections = content?.sections || [];
  const i = sections.findIndex((s) => seconds < (s.section_end_time || 0));
  return i === -1 ? Math.max(0, sections.length - 1) : i;
}

// A transcript line after which the recording leaves silence to read the
// questions or check answers ("you have some time to look at questions 1
// to 6", "half a minute to check your answers").
const PAUSE_PROMPT = /(some time|seconds|half a minute|a minute)\s+to\s+(look|read|check)/i;
const PAUSE_SECONDS = 30;
// Start a little before the estimate: early only costs a few seconds of
// listening, late misses the answer.
const LEAD_SECONDS = 10;

// hasQuestionTimes reports whether a section's questions carry their own
// timestamp_hint. Many imported tests give every question the same one,
// or one per question group (where the questions start), which is no use
// for replaying one answer. (A pair answered together may share one.)
export function hasQuestionTimes(section) {
  const hints = (section?.question_groups || []).flatMap((g) => g.questions.map((q) => q.timestamp_hint));
  return hints.length > 0 && hints.every((h) => h != null) && new Set(hints).size >= 0.7 * hints.length;
}

// estimateTime places a point in a section's transcript (paragraph, and
// character offset within it) on the recording, assuming steady speech
// across the section's span apart from the pauses the transcript
// announces. Null without the section's span.
export function estimateTime(section, paragraphs, paragraph, offset = 0) {
  const start = section?.section_start_time || 0;
  const end = section?.section_end_time;
  if (!end || end <= start || !paragraphs?.length) return null;
  const lengths = paragraphs.map((p) => (p.text || '').length);
  const pauses = paragraphs.map((p) => (PAUSE_PROMPT.test(p.text || '') ? PAUSE_SECONDS : 0));
  const sum = (xs) => xs.reduce((a, b) => a + b, 0);
  const speech = Math.max(1, end - start - sum(pauses));
  const perChar = speech / Math.max(1, sum(lengths));
  const before = sum(lengths.slice(0, paragraph)) + offset;
  const t = start + before * perChar + sum(pauses.slice(0, paragraph)) - LEAD_SECONDS;
  return Math.min(Math.max(start, t), end);
}
