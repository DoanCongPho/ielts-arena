// Mirrors internal/feature/ielts_test/models.go's QuestionType catalogue and
// per-skill allow-lists, plus a UI-only "archetype" classification that
// decides which fields the test-builder form shows for each type.

export const READING_TYPES = [
  'true-false-not-given',
  'yes-no-not-given',
  'multiple-choice',
  'multiple-choice-multi',
  'matching-headings',
  'matching-information',
  'matching-features',
  'matching-sentence-endings',
  'sentence-completion',
  'summary-completion',
  'table-completion',
  'short-answer',
  'diagram-label-completion',
  'flow-chart-completion',
  'note-completion',
];

export const LISTENING_TYPES = [
  'form-completion',
  'note-completion',
  'table-completion',
  'flow-chart-completion',
  'summary-completion',
  'sentence-completion',
  'multiple-choice',
  'multiple-choice-multi',
  'matching',
  'map-plan-labelling',
];

export const QUESTION_TYPE_LABELS = {
  'true-false-not-given': 'True / False / Not Given',
  'yes-no-not-given': 'Yes / No / Not Given',
  'multiple-choice': 'Multiple choice (1 đáp án)',
  'multiple-choice-multi': 'Multiple choice (chọn nhiều)',
  'matching-headings': 'Matching headings',
  'matching-information': 'Matching information',
  'matching-features': 'Matching features',
  'matching-sentence-endings': 'Matching sentence endings',
  'sentence-completion': 'Sentence completion',
  'summary-completion': 'Summary completion',
  'table-completion': 'Table completion',
  'short-answer': 'Short answer questions',
  'diagram-label-completion': 'Diagram label completion',
  'flow-chart-completion': 'Flow-chart completion',
  'note-completion': 'Note completion',
  'form-completion': 'Form completion',
  matching: 'Matching (ý kiến/đặc điểm)',
  'map-plan-labelling': 'Map/plan labelling',
};

// CHOICE: each question is answered by picking one/many keys — either from
// the group's shared_options (matching-*, map-plan-labelling) or from that
// question's own options (multiple-choice, multiple-choice-multi).
export const CHOICE_TYPES = new Set([
  'true-false-not-given',
  'yes-no-not-given',
  'multiple-choice',
  'multiple-choice-multi',
  'matching-headings',
  'matching-information',
  'matching-features',
  'matching-sentence-endings',
  'matching',
  'map-plan-labelling',
]);

// PLAIN: each question has its own text prompt and a free-text answer.
export const PLAIN_TYPES = new Set(['sentence-completion', 'short-answer', 'diagram-label-completion']);

// STRUCTURED: questions have no text of their own — they answer a
// positional "{{gap}}" inside the group's shared structure.
export const STRUCTURED_TYPES = new Set(['summary-completion', 'table-completion', 'note-completion', 'flow-chart-completion', 'form-completion']);

// Types where the group provides a fixed answer vocabulary (shown as a
// dropdown) rather than free per-question options.
export const SHARED_OPTIONS_TYPES = new Set([
  'matching-headings',
  'matching-information',
  'matching-features',
  'matching-sentence-endings',
  'matching',
]);

export const TFNG_ANSWERS = ['TRUE', 'FALSE', 'NOT GIVEN'];
export const YNNG_ANSWERS = ['YES', 'NO', 'NOT GIVEN'];

export function newOption(id = '', text = '') {
  return { id, text };
}

export function emptyQuestionGroup(questionType) {
  return {
    question_type: questionType,
    instructions: '',
    questions: [],
    shared_options: [],
    select_count: 2,
    allow_reuse: false,
    word_limit: null,
    has_word_bank: false,
    word_bank: [],
    summary_text: '',
    table_structure: { columns: [''], rows: [['']] },
    note_structure: { title: '', items: [''] },
    flow_structure: { steps: [''] },
    form_structure: { title: '', fields: [''] },
    diagram_image_url: '',
    map_image_url: '',
    location_key: [],
  };
}

export function emptyQuestion(questionType) {
  if (CHOICE_TYPES.has(questionType)) {
    return { text: '', options: [], answer: '' };
  }
  if (PLAIN_TYPES.has(questionType)) {
    return { text: '', answer: '', accepted_answers: '' };
  }
  return { answer: '', accepted_answers: '' };
}

// countGaps counts literal "{{gap}}" markers across one or more strings —
// used to show authors a live "N gaps vs M questions" hint before they submit.
export function countGaps(...texts) {
  return texts.reduce((total, t) => total + (String(t || '').match(/\{\{gap\}\}/g) || []).length, 0);
}
