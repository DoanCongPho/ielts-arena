import GroupsListEditor from './GroupsListEditor';
import { READING_TYPES } from '../../lib/questionTypes';

function emptyPassage() {
  return { title: '', paragraphs: [{ label: '', text: '' }], question_groups: [] };
}

export default function PassagesEditor({ passages, onChange }) {
  function updatePassage(i, patch) {
    onChange(passages.map((p, idx) => (idx === i ? { ...p, ...patch } : p)));
  }

  function addPassage() {
    onChange([...passages, emptyPassage()]);
  }

  function removePassage(i) {
    onChange(passages.filter((_, idx) => idx !== i));
  }

  function updateParagraph(pi, li, field, value) {
    const paragraphs = passages[pi].paragraphs.map((p, idx) => (idx === li ? { ...p, [field]: value } : p));
    updatePassage(pi, { paragraphs });
  }

  function addParagraph(pi) {
    updatePassage(pi, { paragraphs: [...passages[pi].paragraphs, { label: '', text: '' }] });
  }

  function removeParagraph(pi, li) {
    updatePassage(pi, { paragraphs: passages[pi].paragraphs.filter((_, idx) => idx !== li) });
  }

  return (
    <div className="tb-passages">
      {passages.map((p, pi) => (
        <div key={pi} className="tb-passage-card">
          <div className="tb-passage-header">
            <input
              className="tb-input tb-input-grow"
              placeholder={`Tiêu đề Passage ${pi + 1}`}
              value={p.title}
              onChange={(e) => updatePassage(pi, { title: e.target.value })}
            />
            <button type="button" className="tb-remove-btn" onClick={() => removePassage(pi)}>
              ✕ Xoá passage
            </button>
          </div>

          <div className="tb-field">
            <span>Đoạn văn (paragraphs)</span>
            {p.paragraphs.map((para, li) => (
              <div key={li} className="tb-paragraph-row">
                <input
                  className="tb-input tb-input-key"
                  placeholder="Nhãn (A, B...)"
                  value={para.label}
                  onChange={(e) => updateParagraph(pi, li, 'label', e.target.value)}
                />
                <textarea
                  className="tb-textarea tb-input-grow"
                  rows={3}
                  placeholder="Nội dung đoạn văn"
                  value={para.text}
                  onChange={(e) => updateParagraph(pi, li, 'text', e.target.value)}
                />
                <button type="button" className="tb-remove-btn" onClick={() => removeParagraph(pi, li)}>
                  ✕
                </button>
              </div>
            ))}
            <button type="button" className="tb-add-btn" onClick={() => addParagraph(pi)}>
              + Thêm đoạn văn
            </button>
          </div>

          <div className="tb-field">
            <span>Câu hỏi</span>
            <GroupsListEditor
              groups={p.question_groups}
              allowedTypes={READING_TYPES}
              onChange={(v) => updatePassage(pi, { question_groups: v })}
            />
          </div>
        </div>
      ))}

      <button type="button" className="tb-add-btn tb-add-btn-big" onClick={addPassage}>
        + Thêm Passage
      </button>
    </div>
  );
}
