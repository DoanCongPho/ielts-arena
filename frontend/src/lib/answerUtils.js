// isAnswered reports whether a live-mode answer value (a plain string for
// single-answer question types, or an array for multiple-choice-multi) has
// actually been filled in.
export function isAnswered(value) {
  if (Array.isArray(value)) return value.length > 0;
  return (value || '').toString().trim() !== '';
}

// questionSpan mirrors the backend's questionSpan (autograde.go): a
// multiple-choice-multi question covers select_count consecutive question
// numbers and is worth that many marks ("Questions 23-24: choose TWO"), with
// question_order as the first of them. Everything else covers one.
export function questionSpan(group) {
  return group.question_type === 'multiple-choice-multi' && group.select_count > 1 ? group.select_count : 1;
}

// questionNumberLabel renders a question's number, as a range when it spans
// several ("23-24").
export function questionNumberLabel(order, span = 1) {
  return span > 1 ? `${order}-${order + span - 1}` : String(order);
}

// flattenQuestions lists every question across groups in document order,
// tagging each with its `span` so group-less consumers (the nav bar) can
// number and count them by marks.
export function flattenQuestions(groups) {
  return (groups || []).flatMap((g) => g.questions.map((q) => ({ ...q, span: questionSpan(g) })));
}
