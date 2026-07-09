import OptionListEditor from './OptionListEditor';
import StringListEditor from './StringListEditor';
import TableStructureEditor from './TableStructureEditor';
import {
  QUESTION_TYPE_LABELS,
  CHOICE_TYPES,
  PLAIN_TYPES,
  STRUCTURED_TYPES,
  SHARED_OPTIONS_TYPES,
  TFNG_ANSWERS,
  YNNG_ANSWERS,
  emptyQuestion,
  countGaps,
} from '../../lib/questionTypes';

// QuestionGroupEditor is the type-dispatching form for one question_group:
// the shared fields every type has (instructions), the type-specific extra
// fields (shared_options / word_bank / table_structure / ...), and the
// nested list of questions, each rendered according to the type's answer
// archetype (choice / plain-text / structured-gap).
export default function QuestionGroupEditor({ group, allowedTypes, onChange, onRemove }) {
  function set(field, value) {
    onChange({ ...group, [field]: value });
  }

  function setQuestion(i, patch) {
    const next = group.questions.map((q, idx) => (idx === i ? { ...q, ...patch } : q));
    onChange({ ...group, questions: next });
  }

  function addQuestion() {
    onChange({ ...group, questions: [...group.questions, emptyQuestion(group.question_type)] });
  }

  function removeQuestion(i) {
    onChange({ ...group, questions: group.questions.filter((_, idx) => idx !== i) });
  }

  const isChoice = CHOICE_TYPES.has(group.question_type);
  const isPlain = PLAIN_TYPES.has(group.question_type);
  const isStructured = STRUCTURED_TYPES.has(group.question_type);
  const usesSharedOptions = SHARED_OPTIONS_TYPES.has(group.question_type);
  const isMulti = group.question_type === 'multiple-choice-multi';
  const isMapPlan = group.question_type === 'map-plan-labelling';

  return (
    <div className="tb-group-card">
      <div className="tb-group-header">
        <select
          className="tb-select"
          value={group.question_type}
          onChange={(e) => onChange({ ...emptyQuestionGroupPreserving(group), question_type: e.target.value })}
        >
          {allowedTypes.map((t) => (
            <option key={t} value={t}>
              {QUESTION_TYPE_LABELS[t]}
            </option>
          ))}
        </select>
        <button type="button" className="tb-remove-btn" onClick={onRemove}>
          ✕ Xoá nhóm
        </button>
      </div>

      <label className="tb-field">
        <span>Hướng dẫn (instructions)</span>
        <textarea
          className="tb-textarea"
          rows={2}
          value={group.instructions}
          onChange={(e) => set('instructions', e.target.value)}
        />
      </label>

      {isMulti && (
        <label className="tb-field tb-field-inline">
          <span>Số đáp án cần chọn (select_count)</span>
          <input
            type="number"
            min={2}
            className="tb-input tb-input-narrow"
            value={group.select_count}
            onChange={(e) => set('select_count', Number(e.target.value))}
          />
        </label>
      )}

      {(usesSharedOptions || isMapPlan) && (
        <div className="tb-field">
          <span>{isMapPlan ? 'Chú giải vị trí trên bản đồ (location_key)' : 'Danh sách lựa chọn dùng chung (shared_options)'}</span>
          <OptionListEditor
            options={isMapPlan ? group.location_key : group.shared_options}
            onChange={(v) => set(isMapPlan ? 'location_key' : 'shared_options', v)}
          />
        </div>
      )}

      {isMapPlan && (
        <label className="tb-field">
          <span>URL ảnh bản đồ (map_image_url) — bắt buộc</span>
          <input className="tb-input" value={group.map_image_url} onChange={(e) => set('map_image_url', e.target.value)} />
        </label>
      )}

      {(group.question_type === 'matching-information' || group.question_type === 'matching-features') && (
        <label className="tb-field tb-field-checkbox">
          <input
            type="checkbox"
            checked={!!group.allow_reuse}
            onChange={(e) => set('allow_reuse', e.target.checked)}
          />
          <span>Cho phép dùng lại 1 lựa chọn cho nhiều câu (allow_reuse)</span>
        </label>
      )}

      {(group.question_type === 'sentence-completion' || group.question_type === 'short-answer') && (
        <label className="tb-field tb-field-inline">
          <span>Giới hạn số từ (word_limit, tuỳ chọn)</span>
          <input
            type="number"
            min={0}
            className="tb-input tb-input-narrow"
            value={group.word_limit ?? ''}
            onChange={(e) => set('word_limit', e.target.value === '' ? null : Number(e.target.value))}
          />
        </label>
      )}

      {group.question_type === 'diagram-label-completion' && (
        <label className="tb-field">
          <span>URL ảnh sơ đồ (diagram_image_url) — bắt buộc</span>
          <input className="tb-input" value={group.diagram_image_url} onChange={(e) => set('diagram_image_url', e.target.value)} />
        </label>
      )}

      {group.question_type === 'summary-completion' && (
        <>
          <label className="tb-field tb-field-checkbox">
            <input
              type="checkbox"
              checked={!!group.has_word_bank}
              onChange={(e) => set('has_word_bank', e.target.checked)}
            />
            <span>Có word bank để chọn (has_word_bank)</span>
          </label>
          {group.has_word_bank && (
            <div className="tb-field">
              <span>Word bank</span>
              <OptionListEditor options={group.word_bank} onChange={(v) => set('word_bank', v)} />
            </div>
          )}
          <label className="tb-field">
            <span>Đoạn tóm tắt (summary_text) — dùng {'{{gap}}'} cho mỗi chỗ trống</span>
            <textarea
              className="tb-textarea"
              rows={4}
              value={group.summary_text}
              onChange={(e) => set('summary_text', e.target.value)}
            />
          </label>
          <GapHint count={countGaps(group.summary_text)} questionCount={group.questions.length} />
        </>
      )}

      {group.question_type === 'table-completion' && (
        <div className="tb-field">
          <span>Bảng (table_structure) — điền {'{{gap}}'} vào ô trống</span>
          <TableStructureEditor value={group.table_structure} onChange={(v) => set('table_structure', v)} />
          <GapHint
            count={(group.table_structure.rows || []).reduce((n, row) => n + countGaps(...row), 0)}
            questionCount={group.questions.length}
          />
        </div>
      )}

      {group.question_type === 'note-completion' && (
        <div className="tb-field">
          <span>Ghi chú (note_structure)</span>
          <input
            className="tb-input"
            placeholder="Tiêu đề (tuỳ chọn)"
            value={group.note_structure.title}
            onChange={(e) => set('note_structure', { ...group.note_structure, title: e.target.value })}
          />
          <StringListEditor
            items={group.note_structure.items}
            onChange={(v) => set('note_structure', { ...group.note_structure, items: v })}
          />
          <GapHint count={countGaps(...group.note_structure.items)} questionCount={group.questions.length} />
        </div>
      )}

      {group.question_type === 'flow-chart-completion' && (
        <div className="tb-field">
          <span>Các bước (flow_structure.steps)</span>
          <StringListEditor items={group.flow_structure.steps} onChange={(v) => set('flow_structure', { steps: v })} />
          <GapHint count={countGaps(...group.flow_structure.steps)} questionCount={group.questions.length} />
        </div>
      )}

      {group.question_type === 'form-completion' && (
        <div className="tb-field">
          <span>Biểu mẫu (form_structure)</span>
          <input
            className="tb-input"
            placeholder="Tiêu đề (tuỳ chọn)"
            value={group.form_structure.title}
            onChange={(e) => set('form_structure', { ...group.form_structure, title: e.target.value })}
          />
          <StringListEditor
            items={group.form_structure.fields}
            onChange={(v) => set('form_structure', { ...group.form_structure, fields: v })}
          />
          <GapHint count={countGaps(...group.form_structure.fields)} questionCount={group.questions.length} />
        </div>
      )}

      <div className="tb-questions">
        <span className="tb-questions-label">Câu hỏi ({group.questions.length})</span>
        {group.questions.map((q, i) => (
          <QuestionRow
            key={i}
            index={i}
            question={q}
            group={group}
            isChoice={isChoice}
            isPlain={isPlain}
            isStructured={isStructured}
            usesSharedOptions={usesSharedOptions || isMapPlan}
            isMulti={isMulti}
            onChange={(patch) => setQuestion(i, patch)}
            onRemove={() => removeQuestion(i)}
          />
        ))}
        <button type="button" className="tb-add-btn" onClick={addQuestion}>
          + Thêm câu hỏi
        </button>
      </div>
    </div>
  );
}

function emptyQuestionGroupPreserving(group) {
  // Switching question_type keeps instructions but resets type-specific
  // fields/questions, since e.g. a table_structure doesn't carry over to
  // multiple-choice.
  return { ...group, questions: [] };
}

function GapHint({ count, questionCount }) {
  const ok = count === questionCount;
  return (
    <p className={`tb-gap-hint ${ok ? 'tb-gap-hint-ok' : 'tb-gap-hint-bad'}`}>
      {count} chỗ trống {'{{gap}}'} — {questionCount} câu hỏi {ok ? '✓ khớp' : '✗ phải bằng nhau'}
    </p>
  );
}

function QuestionRow({ index, question, group, isChoice, isPlain, isStructured, usesSharedOptions, isMulti, onChange, onRemove }) {
  const optionsSource = usesSharedOptions
    ? group.question_type === 'map-plan-labelling'
      ? group.location_key
      : group.shared_options
    : question.options || [];

  function toggleMultiAnswer(id) {
    const current = Array.isArray(question.answer) ? question.answer : [];
    if (current.includes(id)) {
      onChange({ answer: current.filter((x) => x !== id) });
    } else if (current.length < (group.select_count || 99)) {
      onChange({ answer: [...current, id] });
    }
  }

  return (
    <div className="tb-question-row">
      <div className="tb-question-row-header">
        <span className="tb-question-index">#{index + 1}</span>
        <button type="button" className="tb-remove-btn" onClick={onRemove}>
          ✕
        </button>
      </div>

      {(isChoice || isPlain) && (
        <input
          className="tb-input"
          placeholder="Nội dung câu hỏi (text)"
          value={question.text}
          onChange={(e) => onChange({ text: e.target.value })}
        />
      )}

      {isChoice && !usesSharedOptions && (
        <div className="tb-field">
          <span>Lựa chọn riêng cho câu này (options)</span>
          <OptionListEditor idLabel="Key (A, B...)" options={question.options || []} onChange={(v) => onChange({ options: v })} />
        </div>
      )}

      {isChoice && group.question_type === 'true-false-not-given' && (
        <AnswerSelect value={question.answer} options={TFNG_ANSWERS} onChange={(v) => onChange({ answer: v })} />
      )}

      {isChoice && group.question_type === 'yes-no-not-given' && (
        <AnswerSelect value={question.answer} options={YNNG_ANSWERS} onChange={(v) => onChange({ answer: v })} />
      )}

      {isChoice && isMulti && (
        <div className="tb-field">
          <span>Đáp án đúng (chọn {group.select_count})</span>
          <div className="tb-checkbox-list">
            {optionsSource.map((opt) => (
              <label key={opt.id} className="tb-field-checkbox">
                <input
                  type="checkbox"
                  checked={Array.isArray(question.answer) && question.answer.includes(opt.id)}
                  onChange={() => toggleMultiAnswer(opt.id)}
                />
                <span>{opt.id ? `${opt.id}. ${opt.text}` : opt.text}</span>
              </label>
            ))}
          </div>
        </div>
      )}

      {isChoice && !isMulti && group.question_type !== 'true-false-not-given' && group.question_type !== 'yes-no-not-given' && (
        <AnswerSelect
          value={question.answer}
          options={optionsSource.map((o) => o.id)}
          labels={optionsSource}
          onChange={(v) => onChange({ answer: v })}
          placeholder="-- Chọn đáp án đúng --"
        />
      )}

      {isPlain && (
        <>
          <input
            className="tb-input"
            placeholder="Đáp án chuẩn (answer)"
            value={question.answer}
            onChange={(e) => onChange({ answer: e.target.value })}
          />
          <input
            className="tb-input"
            placeholder="Các cách viết chấp nhận, cách nhau bằng dấu phẩy (accepted_answers)"
            value={question.accepted_answers}
            onChange={(e) => onChange({ accepted_answers: e.target.value })}
          />
        </>
      )}

      {isStructured && group.question_type === 'summary-completion' && group.has_word_bank && (
        <AnswerSelect
          value={question.answer}
          options={(group.word_bank || []).map((o) => o.id)}
          labels={group.word_bank}
          onChange={(v) => onChange({ answer: v })}
          placeholder="-- Chọn từ word bank --"
        />
      )}

      {isStructured && !(group.question_type === 'summary-completion' && group.has_word_bank) && (
        <>
          <input
            className="tb-input"
            placeholder={`Đáp án chuẩn cho chỗ trống #${index + 1} (answer)`}
            value={question.answer}
            onChange={(e) => onChange({ answer: e.target.value })}
          />
          <input
            className="tb-input"
            placeholder="Các cách viết chấp nhận, cách nhau bằng dấu phẩy (accepted_answers)"
            value={question.accepted_answers}
            onChange={(e) => onChange({ accepted_answers: e.target.value })}
          />
        </>
      )}
    </div>
  );
}

function AnswerSelect({ value, options, labels, onChange, placeholder = '-- Chọn --' }) {
  return (
    <select className="tb-select" value={value || ''} onChange={(e) => onChange(e.target.value)}>
      <option value="">{placeholder}</option>
      {options.map((opt, i) => (
        <option key={opt} value={opt}>
          {labels ? `${labels[i]?.id ?? opt}. ${labels[i]?.text ?? ''}` : opt}
        </option>
      ))}
    </select>
  );
}
