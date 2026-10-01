import './HighlightToolbar.css';

// HighlightToolbar is the floating "Tô đậm" button over a text selection
// (see useTextHighlights).
// A review labels it as saving the word to the vocabulary list.
export default function HighlightToolbar({ selection, onApply, toolbarRef, label = '🖍 Tô đậm' }) {
  if (!selection) return null;
  return (
    <button
      ref={toolbarRef}
      type="button"
      className="reading-highlight-toolbar"
      style={{ left: selection.x, top: selection.y }}
      onClick={onApply}
    >
      {label}
    </button>
  );
}
