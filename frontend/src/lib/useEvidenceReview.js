import { useEffect, useState } from 'react';
import { getAnswerKey } from './api';
import { evidenceRanges } from './evidence';

// useEvidenceReview powers a graded reading review: once `enabled` it loads
// the test's answer key (explanations + evidence), and locate(order) moves
// to the passage holding that question, highlights the text its evidence
// cites and scrolls to it. evidenceFor(passage, paragraph) gives the
// highlight ranges for one paragraph, to pass to HighlightableText.
// selectPassage is the page's own passage-tab setter.
export function useEvidenceReview(testId, enabled, passages, selectPassage) {
  const [answerKey, setAnswerKey] = useState(null);
  const [focus, setFocus] = useState(null);

  useEffect(() => {
    if (!enabled || !testId) return;
    let cancelled = false;
    getAnswerKey(testId)
      .then((key) => {
        if (!cancelled) setAnswerKey(key);
      })
      // Explanations are extra: without them the review still shows the
      // marked answers, so a failed fetch just leaves them out.
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [testId, enabled]);

  // Scroll once the (possibly newly selected) passage has rendered the mark.
  useEffect(() => {
    if (!focus) return;
    const mark = document.querySelector(`#passage-${focus.passage}-p-${focus.paragraph} .reading-evidence-mark`);
    mark?.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }, [focus]);

  function locate(order) {
    const entry = answerKey?.[order];
    const passage = passages.findIndex((p) =>
      (p.question_groups || []).some((g) => g.questions.some((q) => q.question_order === order)),
    );
    if (passage === -1 || !entry?.evidence?.length) return;
    const ranges = evidenceRanges(passages[passage].paragraphs, entry.evidence);
    const found = Object.keys(ranges).map(Number);
    if (found.length === 0) return;
    selectPassage(passage);
    // A fresh object each time, so locating the same question again still
    // re-runs the scroll effect.
    setFocus({ passage, ranges, paragraph: Math.min(...found) });
  }

  function evidenceFor(passage, paragraph) {
    return focus?.passage === passage ? focus.ranges[paragraph] : undefined;
  }

  return { answerKey, locate, evidenceFor };
}
