import { optionLabel } from './chipPicker';

// The pieces shared by every "put an option from a list into a slot"
// question: matching and map labelling (MatchingDragDrop), and summaries
// completed from a word list (StructuredBlankGroup). An option is dragged
// from the bank onto a slot, or picked (click, tap, Enter/Space) and then
// placed by clicking the slot — the way in on phones, where HTML5 drag
// and drop doesn't work.
//
// Chips and slots are spans with role="button", not <button>s: Safari
// won't start a drag from a <button>, and a slot can sit inside a
// sentence (<p>), where only phrasing content is valid.

// onActivate makes a role="button" span answer Enter and Space like a
// real button.
function onActivate(fn) {
  return (e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      fn();
    }
  };
}

// ChipBank lists the options. One already placed is struck through, so
// the ones left stand out; a single-use one can't be placed again, and
// with removeUsed it leaves the bank until its slot is cleared. A review
// (disabled) always shows the whole bank.
export function ChipBank({ options, picker, usedIds, singleUse, removeUsed }) {
  return (
    <div className="matching-dnd-bank">
      {options.map((opt) => {
        const used = usedIds.has(opt.id);
        const locked = used && singleUse;
        if (locked && removeUsed && !picker.disabled) return null;
        const inactive = picker.disabled || locked;
        const pick = () => !inactive && picker.toggle(opt.id);
        return (
          <span
            key={opt.id}
            role="button"
            tabIndex={inactive ? -1 : 0}
            aria-disabled={inactive || undefined}
            aria-pressed={picker.selected === opt.id}
            draggable={!inactive}
            className={`matching-chip ${picker.selected === opt.id ? 'matching-chip-selected' : ''} ${used ? 'matching-chip-used' : ''} ${inactive ? 'matching-chip-inactive' : ''}`}
            onDragStart={(e) => {
              e.dataTransfer.effectAllowed = 'move';
              e.dataTransfer.setData('text/plain', opt.id);
            }}
            onClick={pick}
            onKeyDown={onActivate(pick)}
          >
            {optionLabel(opt)}
          </span>
        );
      })}
    </div>
  );
}

// DropSlot is where one question's option goes: a row under the question
// (matching), or inline in the text (a summary gap).
export function DropSlot({ id, order, chosen, picker, inline, stateClass = '', placeholder }) {
  const { disabled } = picker;
  const place = () => {
    if (picker.selected) picker.assign(order, picker.selected);
  };
  // Safari only fires drop when dragenter, as well as dragover, is
  // cancelled.
  const accept = (e) => {
    if (disabled) return;
    e.preventDefault();
    picker.setOverOrder(order);
  };
  return (
    <span
      id={id}
      role="button"
      tabIndex={disabled ? -1 : 0}
      aria-label={`Câu ${order}: ${chosen ? optionLabel(chosen) : 'chưa chọn'}`}
      className={`${inline ? 'gap-drop-slot' : 'matching-drop-slot'} ${picker.overOrder === order ? 'matching-drop-slot-over' : ''} ${chosen ? 'matching-drop-slot-filled' : ''} ${stateClass}`}
      onDragEnter={accept}
      onDragOver={accept}
      onDragLeave={() => picker.setOverOrder((o) => (o === order ? null : o))}
      onDrop={(e) => {
        e.preventDefault();
        picker.setOverOrder(null);
        const optionId = e.dataTransfer.getData('text/plain');
        if (optionId) picker.assign(order, optionId);
      }}
      onClick={place}
      onKeyDown={onActivate(place)}
    >
      {chosen ? (
        <span className="matching-drop-value">
          {optionLabel(chosen)}
          {!disabled && (
            <button
              type="button"
              className="matching-drop-clear"
              onClick={(e) => {
                e.stopPropagation();
                picker.clear(order);
              }}
              aria-label="Bỏ đáp án"
            >
              ×
            </button>
          )}
        </span>
      ) : (
        <span className="matching-drop-placeholder">{placeholder}</span>
      )}
    </span>
  );
}
