import { CHOICE_TYPES, PLAIN_TYPES, SHARED_OPTIONS_TYPES } from './questionTypes';

function cleanOptions(list) {
  return (list || []).filter((o) => (o.id && o.id.trim()) || (o.text && o.text.trim()));
}

// splitAccepted turns the comma-separated "accepted answers" UI field into
// an array, making sure the canonical answer itself is included — the
// backend requires accepted_answers to contain the answer's own value.
function splitAccepted(acceptedStr, answer) {
  const parts = String(acceptedStr || '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean);
  const answerTrim = String(answer || '').trim();
  if (answerTrim && !parts.some((p) => p.toLowerCase() === answerTrim.toLowerCase())) {
    parts.unshift(answerTrim);
  }
  return parts;
}

function buildQuestion(q, groupType, hasWordBank, orderRef) {
  const question_order = orderRef.next++;

  if (CHOICE_TYPES.has(groupType)) {
    const out = { question_order, answer: q.answer };
    if (q.text) out.text = q.text;
    if (!SHARED_OPTIONS_TYPES.has(groupType) && groupType !== 'map-plan-labelling') {
      const opts = cleanOptions(q.options);
      if (opts.length) out.options = opts;
    }
    return out;
  }

  if (PLAIN_TYPES.has(groupType)) {
    return {
      question_order,
      text: q.text,
      answer: q.answer,
      accepted_answers: splitAccepted(q.accepted_answers, q.answer),
    };
  }

  // Structured (summary/table/note/flow/form completion).
  if (groupType === 'summary-completion' && hasWordBank) {
    return { question_order, answer: q.answer };
  }
  return {
    question_order,
    answer: q.answer,
    accepted_answers: splitAccepted(q.accepted_answers, q.answer),
  };
}

function buildGroup(group, groupOrder, orderRef) {
  const type = group.question_type;
  const out = {
    group_order: groupOrder,
    question_type: type,
    instructions: group.instructions,
    questions: group.questions.map((q) => buildQuestion(q, type, group.has_word_bank, orderRef)),
  };

  if (type === 'multiple-choice-multi') out.select_count = group.select_count;

  if (SHARED_OPTIONS_TYPES.has(type)) {
    out.shared_options = cleanOptions(group.shared_options);
    if (type === 'matching-information' || type === 'matching-features') {
      out.allow_reuse = !!group.allow_reuse;
    }
  }

  if (type === 'map-plan-labelling') {
    out.map_image_url = group.map_image_url;
    out.location_key = cleanOptions(group.location_key);
  }

  if (type === 'sentence-completion' || type === 'short-answer') {
    if (group.word_limit) out.word_limit = group.word_limit;
  }

  if (type === 'diagram-label-completion') {
    out.diagram_image_url = group.diagram_image_url;
  }

  if (type === 'summary-completion') {
    out.has_word_bank = !!group.has_word_bank;
    if (group.has_word_bank) out.word_bank = cleanOptions(group.word_bank);
    out.summary_text = group.summary_text;
  }

  if (type === 'table-completion') {
    out.table_structure = group.table_structure;
  }

  if (type === 'note-completion') {
    out.note_structure = group.note_structure;
  }

  if (type === 'flow-chart-completion') {
    out.flow_structure = group.flow_structure;
  }

  if (type === 'form-completion') {
    out.form_structure = group.form_structure;
  }

  return out;
}

// buildTestPayload assembles the exact CreateTestRequest body the backend
// expects, auto-numbering question_order globally (1..N, in document
// order) so authors never have to manage that numbering by hand.
export function buildTestPayload({ skill, taskType, source, isCurrent, xpGain, thumbnailUrl, writingContent, passages, listeningAudioUrl, sections }) {
  let content_data;

  if (skill === 'writing') {
    content_data = { prompt: writingContent.prompt, image_url: writingContent.image_url || '' };
  } else if (skill === 'reading') {
    const orderRef = { next: 1 };
    content_data = {
      passages: passages.map((p) => ({
        title: p.title,
        paragraphs: p.paragraphs,
        question_groups: p.question_groups.map((g, gi) => buildGroup(g, gi + 1, orderRef)),
      })),
    };
  } else if (skill === 'listening') {
    const orderRef = { next: 1 };
    content_data = {
      audio_url: listeningAudioUrl,
      sections: sections.map((s) => ({
        title: s.title,
        section_start_time: s.section_start_time,
        section_end_time: s.section_end_time,
        question_groups: s.question_groups.map((g, gi) => buildGroup(g, gi + 1, orderRef)),
      })),
    };
  }

  return {
    skill,
    task_type: taskType,
    source,
    is_current: isCurrent,
    xp_gain: xpGain,
    thumbnail_url: thumbnailUrl,
    content_data,
  };
}
