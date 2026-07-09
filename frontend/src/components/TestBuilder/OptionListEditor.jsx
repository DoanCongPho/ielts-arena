// OptionListEditor edits an array of {id, text} — used for shared_options,
// word_bank, location_key, and per-question options.
export default function OptionListEditor({ options, onChange, idLabel = 'Key', textLabel = 'Nội dung' }) {
  function update(i, field, value) {
    const next = options.map((o, idx) => (idx === i ? { ...o, [field]: value } : o));
    onChange(next);
  }

  function add() {
    onChange([...options, { id: '', text: '' }]);
  }

  function remove(i) {
    onChange(options.filter((_, idx) => idx !== i));
  }

  return (
    <div className="tb-option-list">
      {options.map((opt, i) => (
        <div key={i} className="tb-option-row">
          <input
            className="tb-input tb-input-key"
            placeholder={idLabel}
            value={opt.id}
            onChange={(e) => update(i, 'id', e.target.value)}
          />
          <input
            className="tb-input tb-input-grow"
            placeholder={textLabel}
            value={opt.text}
            onChange={(e) => update(i, 'text', e.target.value)}
          />
          <button type="button" className="tb-remove-btn" onClick={() => remove(i)}>
            ✕
          </button>
        </div>
      ))}
      <button type="button" className="tb-add-btn" onClick={add}>
        + Thêm lựa chọn
      </button>
    </div>
  );
}
