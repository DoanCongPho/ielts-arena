import GroupsListEditor from './GroupsListEditor';
import { LISTENING_TYPES } from '../../lib/questionTypes';

function emptySection() {
  return { title: '', section_start_time: 0, section_end_time: 0, question_groups: [] };
}

export default function SectionsEditor({ sections, onChange }) {
  function updateSection(i, patch) {
    onChange(sections.map((s, idx) => (idx === i ? { ...s, ...patch } : s)));
  }

  function addSection() {
    onChange([...sections, emptySection()]);
  }

  function removeSection(i) {
    onChange(sections.filter((_, idx) => idx !== i));
  }

  return (
    <div className="tb-passages">
      {sections.map((s, si) => (
        <div key={si} className="tb-passage-card">
          <span className="tb-card-eyebrow text-label">Section {si + 1}</span>
          <div className="tb-passage-header">
            <input
              className="tb-input tb-input-grow"
              placeholder={`Tiêu đề Section ${si + 1}`}
              value={s.title}
              onChange={(e) => updateSection(si, { title: e.target.value })}
            />
            <button type="button" className="tb-remove-btn" onClick={() => removeSection(si)}>
              ✕ Xoá section
            </button>
          </div>

          <div className="tb-field tb-field-inline-group">
            <label className="tb-field tb-field-inline">
              <span>Bắt đầu (giây)</span>
              <input
                type="number"
                className="tb-input tb-input-narrow"
                value={s.section_start_time}
                onChange={(e) => updateSection(si, { section_start_time: Number(e.target.value) })}
              />
            </label>
            <label className="tb-field tb-field-inline">
              <span>Kết thúc (giây)</span>
              <input
                type="number"
                className="tb-input tb-input-narrow"
                value={s.section_end_time}
                onChange={(e) => updateSection(si, { section_end_time: Number(e.target.value) })}
              />
            </label>
          </div>

          <div className="tb-field">
            <span>Câu hỏi</span>
            <GroupsListEditor
              groups={s.question_groups}
              allowedTypes={LISTENING_TYPES}
              onChange={(v) => updateSection(si, { question_groups: v })}
            />
          </div>
        </div>
      ))}

      <button type="button" className="tb-add-btn tb-add-btn-big" onClick={addSection}>
        + Thêm Section
      </button>
    </div>
  );
}
