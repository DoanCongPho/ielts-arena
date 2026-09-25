// Submissions graded in the background (writing and speaking take from a
// few seconds to several minutes) are remembered here, so the learner can
// leave the attempt page right after handing in and still be told when the
// result is ready — on whatever page they are, even after a reload.
// GradingNotifier does the polling; pages only add to the list.

const KEY = 'ielts:grading-watch';
export const WATCH_EVENT = 'grading:watch';
// Fired with detail {submission} when a watched submission settles, so an
// open list or review page can refresh itself.
export const SETTLED_EVENT = 'grading:settled';

// Stop watching after this long: the worker gives up well before, and a
// stale entry shouldn't poll forever.
const MAX_AGE_MS = 60 * 60 * 1000;

export function watchedGrades() {
  try {
    const list = JSON.parse(localStorage.getItem(KEY) || '[]');
    const fresh = list.filter((w) => Date.now() - w.since < MAX_AGE_MS);
    if (fresh.length !== list.length) save(fresh);
    return fresh;
  } catch {
    return [];
  }
}

function save(list) {
  try {
    localStorage.setItem(KEY, JSON.stringify(list));
  } catch {
    // Storage blocked: the result is still in the submission history.
  }
}

// watchGrading adds a just-submitted submission; label names it in the
// notification ("Writing Task 2").
export function watchGrading(id, label) {
  const list = watchedGrades().filter((w) => w.id !== id);
  list.push({ id, label, since: Date.now() });
  save(list);
  window.dispatchEvent(new Event(WATCH_EVENT));
}

export function forgetGrading(id) {
  save(watchedGrades().filter((w) => w.id !== id));
}
