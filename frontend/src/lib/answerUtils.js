// isAnswered reports whether a live-mode answer value (a plain string for
// single-answer question types, or an array for multiple-choice-multi) has
// actually been filled in.
export function isAnswered(value) {
  if (Array.isArray(value)) return value.length > 0;
  return (value || '').toString().trim() !== '';
}
