// Speaking content in two shapes: the API's (questions are {id, text}
// objects, ids assigned by the server) and the editors' (plain strings).

export const EMPTY_PART1 = { topics: [{ topic: '', questions: ['', '', ''] }] };
export const EMPTY_PART2 = { topic: 'Describe ', bullets: ['', '', ''], explain: 'and explain ', follow_ups: [] };
export const EMPTY_PART3 = { theme: '', questions: ['', '', '', ''] };

const texts = (qs) => (qs || []).map((q) => q.text);
const questions = (strings) => strings.map((s) => s.trim()).filter(Boolean).map((text) => ({ text }));

export function part1FromApi(p) {
  return { topics: (p?.topics || []).map((t) => ({ topic: t.topic, questions: texts(t.questions) })) };
}

export function part1ToApi(p) {
  return { topics: p.topics.map((t) => ({ topic: t.topic.trim(), questions: questions(t.questions) })) };
}

export function part2FromApi(p) {
  return { topic: p?.topic || '', bullets: p?.bullets || [], explain: p?.explain || '', follow_ups: texts(p?.follow_ups) };
}

export function part2ToApi(p) {
  return {
    topic: p.topic.trim(),
    bullets: p.bullets.map((b) => b.trim()).filter(Boolean),
    explain: p.explain.trim(),
    follow_ups: questions(p.follow_ups),
  };
}

export function part3FromApi(p) {
  return { theme: p?.theme || '', questions: texts(p?.questions) };
}

export function part3ToApi(p) {
  return { theme: p.theme.trim(), questions: questions(p.questions) };
}

export const MODES = [
  { key: 'full', label: 'Full test (Part 1 + 2 + 3)', parts: ['part1', 'part2', 'part3'] },
  { key: 'part1', label: 'Chỉ Part 1', parts: ['part1'] },
  { key: 'part2', label: 'Chỉ Part 2', parts: ['part2'] },
  { key: 'part3', label: 'Chỉ Part 3', parts: ['part3'] },
];

export const MODE_LABELS = { full: 'Full test', part1: 'Part 1', part2: 'Part 2', part3: 'Part 3' };

// speakingSummary is the one line a test card shows: the cue card topic,
// or the first topic/theme for part practice.
export function speakingSummary(content) {
  if (!content) return null;
  const title = content.title ? `${content.title} — ` : '';
  if (content.part2) return title + content.part2.topic;
  if (content.part1) return title + content.part1.topics.map((t) => t.topic).join(', ');
  if (content.part3) return title + (content.part3.theme || content.part3.questions[0]?.text);
  return content.title || null;
}
