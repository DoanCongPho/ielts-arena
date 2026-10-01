import { useState } from 'react';

// Helpers for ChipBank.jsx, kept apart so that file exports components only.

export function optionLabel(opt) {
  return opt.id === opt.text ? opt.text : `${opt.id}. ${opt.text}`;
}

// isSingleUse reports whether each option answers at most one question.
// The instructions decide first: "You may use any letter more than once"
// is copied from the paper, while allow_reuse is set by the importer and
// sometimes contradicts it. Then allow_reuse, when set; otherwise an option
// is reusable only when there are fewer options than questions.
export function isSingleUse(group, options) {
  if (/more than once/i.test(group.instructions || '')) return false;
  if (group.allow_reuse != null) return !group.allow_reuse;
  if (!options) return false;
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
