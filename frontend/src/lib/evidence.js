// Helpers for the answer key's evidence quotes (see Evidence in
// internal/feature/ielts_test/models.go).

const QUOTE_FOLD = { '‘': "'", '’': "'", '“': '"', '”': '"' };

// findQuote locates `quote` inside `text` the way the backend validates it
// (normalizeQuote in autograde.go): whitespace runs collapse to one space
// and typographic quotes fold to plain ones. It returns the matching
// {start, end} character range in the original `text`, or null.
export function findQuote(text, quote) {
  const hay = normalizeWithMap(text);
  const needle = normalizeWithMap(quote).chars.join('');
  if (!needle) return null;
  const at = hay.chars.join('').indexOf(needle);
  if (at === -1) return null;
  return { start: hay.map[at], end: hay.map[at + needle.length - 1] + 1 };
}

// normalizeWithMap normalises s and records, for each normalised
// character, the index of the original character it came from.
function normalizeWithMap(s) {
  const chars = [];
  const map = [];
  for (let i = 0; i < (s || '').length; i++) {
    const c = s[i];
    if (/\s/.test(c)) {
      if (chars.length > 0 && chars[chars.length - 1] !== ' ') {
        chars.push(' ');
        map.push(i);
      }
      continue;
    }
    chars.push(QUOTE_FOLD[c] || c);
    map.push(i);
  }
  if (chars[chars.length - 1] === ' ') {
    chars.pop();
    map.pop();
  }
  return { chars, map };
}

// evidenceRanges turns an answer-key entry's evidence into highlight ranges
// per paragraph index: { [paragraph]: [{start, end}] }. Quotes that can't
// be found (the passage changed since) are skipped.
export function evidenceRanges(paragraphs, evidence) {
  const out = {};
  for (const ev of evidence || []) {
    const text = paragraphs?.[ev.paragraph]?.text;
    const range = text && findQuote(text, ev.quote);
    if (range) (out[ev.paragraph] ||= []).push(range);
  }
  return out;
}
