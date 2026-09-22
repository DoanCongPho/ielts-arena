import './HighlightToolbar.css';

// HighlightToolbar is the floating "Tô đậm" button over a text selection
// (see useTextHighlights).
export default function HighlightToolbar({ selection, onApply, toolbarRef }) {
  if (!selection) return null;
  return (
    <button
      ref={toolbarRef}
      type="button"
      className="reading-highlight-toolbar"
      style={{ left: selection.x, top: selection.y }}
      onClick={onApply}
    >
      🖍 Tô đậm
    </button>
  );
}
