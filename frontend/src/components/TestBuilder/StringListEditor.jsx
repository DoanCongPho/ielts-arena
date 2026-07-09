// StringListEditor edits a plain array of strings (note_structure.items,
// flow_structure.steps, form_structure.fields) — each line can contain the
// literal "{{gap}}" marker to place a blank there.
export default function StringListEditor({ items, onChange, placeholder = 'Dùng {{gap}} để đánh dấu chỗ trống' }) {
  function update(i, value) {
    onChange(items.map((it, idx) => (idx === i ? value : it)));
  }

  function add() {
    onChange([...items, '']);
  }

  function remove(i) {
    onChange(items.filter((_, idx) => idx !== i));
  }

  return (
    <div className="tb-string-list">
      {items.map((item, i) => (
        <div key={i} className="tb-string-row">
          <input
            className="tb-input tb-input-grow"
            placeholder={placeholder}
            value={item}
            onChange={(e) => update(i, e.target.value)}
          />
          <button type="button" className="tb-remove-btn" onClick={() => remove(i)}>
            ✕
          </button>
        </div>
      ))}
      <button type="button" className="tb-add-btn" onClick={add}>
        + Thêm dòng
      </button>
    </div>
  );
}
