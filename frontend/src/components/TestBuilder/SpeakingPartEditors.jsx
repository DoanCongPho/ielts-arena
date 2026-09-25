import StringListEditor from './StringListEditor';

// Editors for the three parts of a speaking test, in the editor shape from
// lib/speakingContent.js. Shared by the admin builder and the user's custom
// test composer.

export function Part1Editor({ value, onChange }) {
  function updateTopic(i, patch) {
    onChange({ topics: value.topics.map((t, idx) => (idx === i ? { ...t, ...patch } : t)) });
  }
  return (
    <div className="tb-field">
      {value.topics.map((t, i) => (
        <div key={i} className="tb-group-card">
          <div className="tb-string-row">
            <input
              className="tb-input tb-input-grow"
              placeholder="Chủ đề quen thuộc (vd: Your hometown)"
              value={t.topic}
              onChange={(e) => updateTopic(i, { topic: e.target.value })}
            />
            {value.topics.length > 1 && (
              <button type="button" className="tb-remove-btn" onClick={() => onChange({ topics: value.topics.filter((_, idx) => idx !== i) })}>
                ✕
              </button>
            )}
          </div>
          <StringListEditor
            items={t.questions}
            onChange={(questions) => updateTopic(i, { questions })}
            placeholder="Câu hỏi (vd: Do you like living there?)"
          />
        </div>
      ))}
      {value.topics.length < 3 && (
        <button type="button" className="tb-add-btn" onClick={() => onChange({ topics: [...value.topics, { topic: '', questions: ['', '', ''] }] })}>
          + Thêm chủ đề
        </button>
      )}
    </div>
  );
}

export function Part2Editor({ value, onChange }) {
  const set = (patch) => onChange({ ...value, ...patch });
  return (
    <div className="tb-field">
      <label className="tb-field">
        <span>Đề cue card</span>
        <input
          className="tb-input"
          placeholder="Describe a book you enjoyed reading."
          value={value.topic}
          onChange={(e) => set({ topic: e.target.value })}
        />
      </label>
      <div className="tb-field">
        <span>You should say: (3–4 ý)</span>
        <StringListEditor
          items={value.bullets}
          onChange={(bullets) => set({ bullets })}
          placeholder="what the book was"
        />
      </div>
      <label className="tb-field">
        <span>Dòng cuối</span>
        <input
          className="tb-input"
          placeholder="and explain why you enjoyed it."
          value={value.explain}
          onChange={(e) => set({ explain: e.target.value })}
        />
      </label>
      <div className="tb-field">
        <span>Câu hỏi sau khi nói (tuỳ chọn, tối đa 2)</span>
        <StringListEditor
          items={value.follow_ups}
          onChange={(follow_ups) => set({ follow_ups: follow_ups.slice(0, 2) })}
          placeholder="Do you often read books?"
        />
      </div>
    </div>
  );
}

export function Part3Editor({ value, onChange, needsTheme }) {
  const set = (patch) => onChange({ ...value, ...patch });
  return (
    <div className="tb-field">
      {needsTheme && (
        <label className="tb-field">
          <span>Chủ đề thảo luận</span>
          <input
            className="tb-input"
            placeholder="reading and books"
            value={value.theme}
            onChange={(e) => set({ theme: e.target.value })}
          />
        </label>
      )}
      <div className="tb-field">
        <span>Câu hỏi thảo luận (3–8 câu, mang tính khái quát)</span>
        <StringListEditor
          items={value.questions}
          onChange={(questions) => set({ questions })}
          placeholder="Why do people read less than they used to?"
        />
      </div>
    </div>
  );
}
