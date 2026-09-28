import { useState } from 'react';

// Helpers for ChipBank.jsx, kept apart so that file exports components only.

export function optionLabel(opt) {
  return opt.id === opt.text ? opt.text : `${opt.id}. ${opt.text}`;
}

// isSingleUse reports whether each option answers at most one question.
// allow_reuse says so when set; otherwise, as on the paper, an option is
// reusable only when the instructions say "more than once", or when there
// are fewer options than questions.
export function isSingleUse(group, options) {
  if (group.allow_reuse != null) return !group.allow_reuse;
  if (!options) return false;
  if (/more than once/i.test(group.instructions || '')) return false;
  return options.length >= group.questions.length;
}

// useChipPicker holds the option picked up for placing and the slot a
// drag is over, shared by a group's bank and its slots.
export function useChipPicker({ disabled, onChange }) {
  const [selected, setSelected] = useState(null);
  const [overOrder, setOverOrder] = useState(null);
  return {
    disabled,
    selected,
    overOrder,
    setOverOrder,
    toggle: (id) => setSelected((cur) => (cur === id ? null : id)),
    assign(order, id) {
      if (disabled) return;
      onChange?.(order, id);
      setSelected(null);
    },
    clear(order) {
      if (!disabled) onChange?.(order, '');
    },
  };
}
