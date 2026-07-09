// TableStructureEditor edits table_structure {columns[], rows[][]} for
// table-completion groups. A cell may contain the literal "{{gap}}" marker.
export default function TableStructureEditor({ value, onChange }) {
  const columns = value.columns || [''];
  const rows = value.rows || [['']];

  function updateColumn(i, text) {
    const next = columns.map((c, idx) => (idx === i ? text : c));
    onChange({ ...value, columns: next, rows: rows.map((r) => resizeRow(r, next.length)) });
  }

  function addColumn() {
    const next = [...columns, ''];
    onChange({ ...value, columns: next, rows: rows.map((r) => resizeRow(r, next.length)) });
  }

  function removeColumn(i) {
    const next = columns.filter((_, idx) => idx !== i);
    onChange({ ...value, columns: next, rows: rows.map((r) => r.filter((_, idx) => idx !== i)) });
  }

  function updateCell(ri, ci, text) {
    const next = rows.map((r, ridx) => (ridx === ri ? r.map((c, cidx) => (cidx === ci ? text : c)) : r));
    onChange({ ...value, rows: next });
  }

  function addRow() {
    onChange({ ...value, rows: [...rows, new Array(columns.length).fill('')] });
  }

  function removeRow(ri) {
    onChange({ ...value, rows: rows.filter((_, idx) => idx !== ri) });
  }

  return (
    <div className="tb-table-editor">
      <div className="tb-table-columns">
        {columns.map((col, i) => (
          <div key={i} className="tb-string-row">
            <input
              className="tb-input tb-input-grow"
              placeholder={`Cột ${i + 1}`}
              value={col}
              onChange={(e) => updateColumn(i, e.target.value)}
            />
            <button type="button" className="tb-remove-btn" onClick={() => removeColumn(i)}>
              ✕
            </button>
          </div>
        ))}
        <button type="button" className="tb-add-btn" onClick={addColumn}>
          + Thêm cột
        </button>
      </div>

      <div className="tb-table-rows">
        {rows.map((row, ri) => (
          <div key={ri} className="tb-table-row">
            {row.map((cell, ci) => (
              <input
                key={ci}
                className="tb-input tb-table-cell"
                placeholder="{{gap}} nếu là chỗ trống"
                value={cell}
                onChange={(e) => updateCell(ri, ci, e.target.value)}
              />
            ))}
            <button type="button" className="tb-remove-btn" onClick={() => removeRow(ri)}>
              ✕
            </button>
          </div>
        ))}
        <button type="button" className="tb-add-btn" onClick={addRow}>
          + Thêm hàng
        </button>
      </div>
    </div>
  );
}

function resizeRow(row, length) {
  const next = row.slice(0, length);
  while (next.length < length) next.push('');
  return next;
}
