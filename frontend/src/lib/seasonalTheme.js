import { useSyncExternalStore } from 'react';

// The Christmas theme turns itself on from 1 December to 6 January
// (Epiphany). A learner can switch it on or off at any time with the
// HUD's ❄ toggle; that choice is kept in localStorage and wins over the
// calendar. `?theme=christmas` in the URL previews it for the rest of the
// browser tab's session.

const PREF_KEY = 'ielts.theme.christmas'; // 'on' | 'off'; unset = follow the season
const PREVIEW_KEY = 'ielts.theme.preview';
const CHANGE_EVENT = 'seasonal-theme:change';

export function isChristmasSeason(date = new Date()) {
  const month = date.getMonth(); // 0-based
  const day = date.getDate();
  return month === 11 || (month === 0 && day <= 6);
}

function readPreview() {
  try {
    const param = new URLSearchParams(window.location.search).get('theme');
    if (param === 'christmas') sessionStorage.setItem(PREVIEW_KEY, 'christmas');
    if (param === 'none') sessionStorage.removeItem(PREVIEW_KEY);
    return sessionStorage.getItem(PREVIEW_KEY) === 'christmas';
  } catch {
    return false;
  }
}

function readPref() {
  try {
    return localStorage.getItem(PREF_KEY);
  } catch {
    return null;
  }
}

// '1' when the theme shows: the learner's own choice if they made one,
// otherwise the season (or a URL preview).
function snapshot() {
  const pref = readPref();
  if (pref === 'on') return '1';
  if (pref === 'off') return '0';
  return isChristmasSeason() || readPreview() ? '1' : '0';
}

function subscribe(onChange) {
  window.addEventListener(CHANGE_EVENT, onChange);
  window.addEventListener('storage', onChange);
  return () => {
    window.removeEventListener(CHANGE_EVENT, onChange);
    window.removeEventListener('storage', onChange);
  };
}

export function setChristmasEnabled(enabled) {
  try {
    localStorage.setItem(PREF_KEY, enabled ? 'on' : 'off');
  } catch {
    // Storage blocked: the toggle just won't stick across reloads.
  }
  window.dispatchEvent(new Event(CHANGE_EVENT));
}

// useChristmasTheme reports whether the theme is showing (`active`) and
// lets the learner switch it (`setEnabled`), in or out of season.
export function useChristmasTheme() {
  const active = useSyncExternalStore(subscribe, snapshot, () => '0') === '1';
  return { active, setEnabled: setChristmasEnabled };
}
