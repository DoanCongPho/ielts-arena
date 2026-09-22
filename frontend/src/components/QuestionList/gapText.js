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

// parseLine reads the light markup an importer may put on a structure
// line (see NoteStructure in internal/feature/ielts_test/models.go):
// "## text" is a subheading, "- text" a bullet, one level deeper per
// leading tab ("\t- text"); anything else is a plain line.
export function parseLine(raw) {
  const s = String(raw ?? '');
  const bullet = s.match(/^(\t*)- ([\s\S]*)$/);
  if (bullet) return { kind: 'bullet', level: bullet[1].length, text: bullet[2] };
  if (s.startsWith('## ')) return { kind: 'heading', level: 0, text: s.slice(3) };
  return { kind: 'text', level: 0, text: s };
}

// resolveMarkedLines parses each line's markup, then numbers its gaps in
// order (as resolveGapLines): [{kind, level, segments}] per line.
export function resolveMarkedLines(lines, questions) {
  const parsed = (lines || []).map(parseLine);
  const segments = resolveGapLines(parsed.map((l) => l.text), questions);
  return parsed.map((l, i) => ({ kind: l.kind, level: l.level, segments: segments[i] }));
}

// resolveGapCells is the table counterpart: each cell may hold several
// "\n"-separated marked lines (e.g. a bulleted list). Gaps are numbered
// row-major (left-to-right, top-to-bottom), and top-to-bottom within a
// cell. Returns rows of cells of resolved lines.
export function resolveGapCells(rows, questions) {
  const flat = [];
  rows.forEach((row, ri) =>
    row.forEach((cell, ci) => String(cell ?? '').split('\n').forEach((line) => flat.push({ ri, ci, line }))),
  );
  const resolved = resolveMarkedLines(
    flat.map((f) => f.line),
    questions,
  );
  const out = rows.map((row) => row.map(() => []));
  flat.forEach((f, i) => out[f.ri][f.ci].push(resolved[i]));
  return out;
}
