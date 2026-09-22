import { useEffect, useState } from 'react';
import { getAnswerKey } from './api';
import { evidenceRanges } from './evidence';

// useEvidenceReview powers a graded review: once `enabled` it loads the
// test's answer key (explanations + evidence, and for listening each
// section's transcript), and locate(order) moves to the passage / section
// holding that question, highlights the text its evidence cites and
// scrolls to it. evidenceFor(unit, paragraph) gives the highlight ranges
// for one paragraph, to pass to HighlightableText.
//
// `units` are the passages (reading) or sections (listening), each with
// question_groups; the text evidence points into is unit.paragraphs, or
// for listening the transcript loaded here (units without paragraphs get
// transcripts[i]). selectUnit is the page's own tab setter.
export function useEvidenceReview(testId, enabled, units, selectUnit) {
  const [answerKey, setAnswerKey] = useState(null);
  const [transcripts, setTranscripts] = useState(null);
  const [focus, setFocus] = useState(null);

  useEffect(() => {
    if (!enabled || !testId) return;
    let cancelled = false;
    getAnswerKey(testId)
      .then((key) => {
        if (cancelled) return;
        setAnswerKey(key.questions);
        setTranscripts(key.transcripts || null);
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

  function textOf(unit) {
    return units[unit]?.paragraphs || transcripts?.[unit] || [];
  }

  function locate(order) {
    const entry = answerKey?.[order];
    const passage = units.findIndex((u) =>
      (u.question_groups || []).some((g) => g.questions.some((q) => q.question_order === order)),
    );
    if (passage === -1 || !entry?.evidence?.length) return;
    const ranges = evidenceRanges(textOf(passage), entry.evidence);
    const found = Object.keys(ranges).map(Number);
    if (found.length === 0) return;
    selectUnit(passage);
    // A fresh object each time, so locating the same question again still
    // re-runs the scroll effect.
    setFocus({ passage, ranges, paragraph: Math.min(...found) });
  }

  function evidenceFor(passage, paragraph) {
    return focus?.passage === passage ? focus.ranges[paragraph] : undefined;
  }

  // transcriptFor(section) is a listening section's transcript, once the
  // answer key has loaded (null before, or for reading).
  function transcriptFor(unit) {
    return transcripts?.[unit] || null;
  }

  return { answerKey, locate, evidenceFor, transcriptFor };
}
