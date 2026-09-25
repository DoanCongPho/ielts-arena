export const SKILLS = [
  { key: 'reading', label: 'Reading' },
  { key: 'listening', label: 'Listening' },
  { key: 'writing', label: 'Writing' },
  { key: 'speaking', label: 'Speaking' },
];

const SERIES_NAMES = { cambridge: 'Cambridge' };

// A filter's `query` is sent to GET /api/tests, which filters before it
// paginates — so a filter always sees every matching test, not just the
// ones on the current page.
//
// Filters shared by skills whose tests come from books: "Cambridge" keeps
// tests in a series, "Khác" keeps the rest.
const SERIES_FILTERS = [
  { key: 'all', label: 'Tất cả', query: {} },
  { key: 'cambridge', label: 'Cambridge', query: { series: 'cambridge' } },
  { key: 'other', label: 'Khác', query: { series: 'none' } },
];

// seriesLabel names the book a test belongs to ("Cambridge 20"), or null
// for a test outside any series.
export function seriesLabel(test) {
  if (!test.series) return null;
  const name = SERIES_NAMES[test.series] || test.series;
  return test.volume ? `${name} ${test.volume}` : name;
}

export const SKILL_CONFIG = {
  writing: {
    label: 'Writing',
    enabled: true,
    colorToken: 'writing',
    attemptPath: (id) => `/practice/writing/${id}`,
    taskFilters: [
      { key: 'all', label: 'Tất cả', query: {} },
      { key: 'task1', label: 'Task 1', query: { task_type: 'task1' } },
      { key: 'task2', label: 'Task 2', query: { task_type: 'task2' } },
    ],
    taskTypeLabel: (taskType) => (taskType === 'task1' ? 'Task 1' : 'Task 2'),
    cardSummary: (content) => content?.prompt,
  },
  reading: {
    label: 'Reading',
    enabled: true,
    colorToken: 'reading',
    attemptPath: (id) => `/practice/reading/${id}`,
    taskFilters: SERIES_FILTERS,
    taskTypeLabel: (taskType) => {
      if (taskType.startsWith('passage')) return taskType.replace('passage', 'Passage ');
      if (taskType.startsWith('test')) return taskType.replace('test', 'Test ');
      return taskType;
    },
    // A length-1 test (authored as a standalone passage) shows a short text
    // preview built from its paragraphs, like Writing does; a real
    // multi-passage full test has no single representative snippet, so it
    // shows no preview (tags only).
    cardSummary: (content) => {
      const passages = content?.passages || [];
      if (passages.length !== 1) return null;
      const paragraphs = passages[0].paragraphs || [];
      return paragraphs.map((p) => p.text).join(' ') || null;
    },
  },
  listening: {
    label: 'Listening',
    enabled: true,
    colorToken: 'listening',
    attemptPath: (id) => `/practice/listening/${id}`,
    taskFilters: SERIES_FILTERS,
    taskTypeLabel: (taskType) => {
      if (taskType.startsWith('section')) return taskType.replace('section', 'Section ');
      if (taskType.startsWith('test')) return taskType.replace('test', 'Test ');
      return taskType;
    },
    // No short text preview for audio content, regardless of section count
    // — the card relies on tags/thumbnail only.
    cardSummary: () => null,
  },
  speaking: {
    label: 'Speaking',
    enabled: false,
    colorToken: 'speaking',
    comingSoon: true,
  },
};
