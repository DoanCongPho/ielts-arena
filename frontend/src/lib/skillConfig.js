export const SKILLS = [
  { key: 'reading', label: 'Reading' },
  { key: 'listening', label: 'Listening' },
  { key: 'writing', label: 'Writing' },
  { key: 'speaking', label: 'Speaking' },
];

export const SKILL_CONFIG = {
  writing: {
    label: 'Writing',
    enabled: true,
    attemptPath: (id) => `/practice/writing/${id}`,
    taskFilters: [
      { key: 'all', label: 'Tất cả' },
      { key: 'task1', label: 'Task 1' },
      { key: 'task2', label: 'Task 2' },
    ],
    taskTypeLabel: (taskType) => (taskType === 'task1' ? 'Task 1' : 'Task 2'),
    cardSummary: (content) => content?.prompt,
  },
  reading: {
    label: 'Reading',
    enabled: true,
    attemptPath: (id) => `/practice/reading/${id}`,
    // Category filters, not one pill per numbered edition — "Passage"
    // matches passage1/passage2/... and "Test" matches test1/test2/...
    // (prefix match, see SkillTestsPage.jsx), so new editions don't need a
    // new filter pill added here every time.
    taskFilters: [
      { key: 'all', label: 'Tất cả' },
      { key: 'passage', label: 'Passage' },
      { key: 'test', label: 'Test' },
    ],
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
    attemptPath: (id) => `/practice/listening/${id}`,
    // Category filters — see the equivalent comment on reading's taskFilters.
    taskFilters: [
      { key: 'all', label: 'Tất cả' },
      { key: 'section', label: 'Section' },
      { key: 'test', label: 'Test' },
    ],
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
  },
};
