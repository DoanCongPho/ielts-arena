import { useEffect, useRef, useState } from 'react';
import './Dropdown.css';

function optionLabel(opt) {
  return opt.id === opt.text ? opt.text : `${opt.id}. ${opt.text}`;
}

// Dropdown is a custom (non-native <select>) picker whose option list is
// always positioned below the trigger. Native <select> lets the browser
// flip the menu upward when it decides there isn't enough room below,
// which was confusing here since triggers often sit near the bottom of a
// scrollable panel — this component pins the menu at top:100% always.
export default function Dropdown({
  id,
  value,
  onChange,
  options,
  disabled,
  placeholder = '-- Chọn --',
  disabledOptionIds,
  wrapperClassName = '',
  triggerClassName = '',
}) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef(null);

  useEffect(() => {
    function handleMouseDown(e) {
      if (!rootRef.current?.contains(e.target)) setOpen(false);
    }
    document.addEventListener('mousedown', handleMouseDown);
    return () => document.removeEventListener('mousedown', handleMouseDown);
  }, []);

  const selected = options.find((o) => o.id === value);

  function pick(id) {
    onChange(id);
    setOpen(false);
  }

  // Built entirely from phrasing-content elements (span/button) rather than
  // div/ul/li — this Dropdown can end up nested inside a <p> (e.g. an inline
  // gap in summary-completion text), and div/ul are not valid there; the
  // browser would silently close the <p> early and produce a mismatched DOM.
  return (
    <span id={id} className={`dropdown ${wrapperClassName}`} ref={rootRef}>
      <button
        type="button"
        className={`dropdown-trigger ${triggerClassName}`}
        disabled={disabled}
        onClick={() => setOpen((o) => !o)}
      >
        <span className="dropdown-trigger-label">{selected ? optionLabel(selected) : placeholder}</span>
        <span className="dropdown-caret">▾</span>
      </button>

      {open && !disabled && (
        <span className="dropdown-menu" role="listbox">
          <button type="button" className="dropdown-option" onClick={() => pick('')}>
            {placeholder}
          </button>
          {options.map((opt) => (
            <button
              key={opt.id}
              type="button"
              className="dropdown-option"
              disabled={disabledOptionIds?.has(opt.id) && opt.id !== value}
              onClick={() => pick(opt.id)}
            >
              {optionLabel(opt)}
            </button>
          ))}
        </span>
      )}
    </span>
  );
}
