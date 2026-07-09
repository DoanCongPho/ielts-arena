const GAP_MARKER = '{{gap}}';

// splitGapSegments turns one line of text into alternating text/gap
// segments, e.g. "Name: {{gap}}, aged {{gap}}" ->
// [{type:'text',value:'Name: '}, {type:'gap'}, {type:'text',value:', aged '}, {type:'gap'}].
function splitGapSegments(text) {
  const parts = String(text ?? '').split(GAP_MARKER);
  const segments = [];
  parts.forEach((part, i) => {
    if (part) segments.push({ type: 'text', value: part });
    if (i < parts.length - 1) segments.push({ type: 'gap' });
  });
  return segments;
}

// resolveGapLines assigns each "{{gap}}" marker across an array of lines
// (in order) to the question at the same position in `questions` — the
// i-th gap found anywhere in `lines` maps to `questions[i]`. Returns one
// segment array per line, with gap segments carrying their `question`.
export function resolveGapLines(lines, questions) {
  let cursor = 0;
  return lines.map((line) => {
    const segments = splitGapSegments(line);
    return segments.map((seg) => {
      if (seg.type !== 'gap') return seg;
      const question = questions[cursor];
      cursor += 1;
      return { ...seg, question };
    });
  });
}

// resolveGapRows is resolveGapLines' 2D counterpart for table_structure.rows
// — gaps are numbered row-major (left-to-right, top-to-bottom) across the
// whole table.
export function resolveGapRows(rows, questions) {
  let cursor = 0;
  return rows.map((row) =>
    row.map((cell) => {
      const segments = splitGapSegments(cell);
      return segments.map((seg) => {
        if (seg.type !== 'gap') return seg;
        const question = questions[cursor];
        cursor += 1;
        return { ...seg, question };
      });
    }),
  );
}
