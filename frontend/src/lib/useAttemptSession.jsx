import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import ConfirmDialog from '../components/ui/ConfirmDialog/ConfirmDialog';

// Reading attempts get a fixed time allowance, as does a writing task of
// unknown type.
export const ATTEMPT_SECONDS = 60 * 60;

// A timed writing attempt gets the time the real exam suggests per task.
export const WRITING_SECONDS = { task1: 20 * 60, task2: 40 * 60 };

// A listening attempt lasts as long as its recording plus this, to check
// answers once the last part has played.
export const LISTENING_CHECK_SECONDS = 2 * 60;

const LEAVE_MESSAGE = 'Bài làm chưa nộp sẽ bị mất.';

// useCountdown counts down from totalSeconds while `running`, and calls
// onExpire once when it reaches zero. It measures against the wall clock
// rather than counting ticks, so a throttled background tab doesn't buy
// extra time. The clock starts the first time `running` is true (i.e. once
// the test has loaded) and freezes when it turns false (submitted).
export function useCountdown(totalSeconds, running, onExpire) {
  const [remaining, setRemaining] = useState(totalSeconds);
  const startedAt = useRef(null);
  const expired = useRef(false);
  // Always call the latest onExpire, so it sees current answers/state.
  const onExpireRef = useRef(onExpire);
  onExpireRef.current = onExpire;

  // The allowance can depend on the test (a listening recording's length),
  // so until the clock starts, show whatever it currently is.
  useEffect(() => {
    if (startedAt.current == null) setRemaining(totalSeconds);
  }, [totalSeconds]);

  useEffect(() => {
    if (!running) return;
    if (startedAt.current == null) startedAt.current = Date.now();

    function tick() {
      const left = Math.max(0, totalSeconds - Math.floor((Date.now() - startedAt.current) / 1000));
      setRemaining(left);
      if (left === 0 && !expired.current) {
        expired.current = true;
        onExpireRef.current?.();
      }
    }
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [running, totalSeconds]);

  return remaining;
}

// useElapsed counts seconds up from the first time `running` is true and
// freezes when it turns false — the untimed counterpart of useCountdown,
// measured against the wall clock for the same reason.
export function useElapsed(running) {
  const [elapsed, setElapsed] = useState(0);
  const startedAt = useRef(null);

  useEffect(() => {
    if (!running) return;
    if (startedAt.current == null) startedAt.current = Date.now();

    function tick() {
      setElapsed(Math.floor((Date.now() - startedAt.current) / 1000));
    }
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [running]);

  return elapsed;
}

// useLeaveGuard stops an in-progress attempt from being left by accident —
// the page's back arrow, a trackpad swipe-back, the browser Back button, a
// reload or closing the tab. While `active`:
//   - horizontal overscroll is disabled, so a two-finger swipe doesn't start
//     the browser's back gesture in the first place;
//   - an extra history entry for the current page is pushed, so Back lands
//     on it instead of leaving; the trap is re-armed at once and the leave
//     dialog opens;
//   - reload/close shows the browser's native "leave site?" prompt (the
//     only thing a browser allows there).
// It returns requestLeave(), for in-app exits (the back arrow): it goes to
// `backTo` straight away when nothing is in progress, and otherwise opens
// the dialog; and leaveDialog, the element the page renders.
// `detail` is the dialog's body, e.g. how many questions are answered.
export function useLeaveGuard(active, { backTo, detail } = {}) {
  const navigate = useNavigate();
  const leaving = useRef(false);
  const [asking, setAsking] = useState(false);

  useEffect(() => {
    if (!active) {
      setAsking(false);
      return;
    }

    const root = document.documentElement;
    const prevOverscroll = root.style.overscrollBehaviorX;
    root.style.overscrollBehaviorX = 'none';

    // Re-use the router's own history state so React Router treats the
    // extra entry as the same location.
    window.history.pushState(window.history.state, '', window.location.href);

    function onPopState() {
      if (leaving.current) return;
      window.history.pushState(window.history.state, '', window.location.href);
      setAsking(true);
    }

    function onBeforeUnload(e) {
      e.preventDefault();
      e.returnValue = '';
    }

    window.addEventListener('popstate', onPopState);
    window.addEventListener('beforeunload', onBeforeUnload);
    return () => {
      root.style.overscrollBehaviorX = prevOverscroll;
      window.removeEventListener('popstate', onPopState);
      window.removeEventListener('beforeunload', onBeforeUnload);
    };
  }, [active]);

  const leave = useCallback(() => {
    leaving.current = true;
    setAsking(false);
    // Replace the trap entry, so Back from the list doesn't land on it.
    navigate(backTo, { replace: active });
  }, [navigate, backTo, active]);

  const requestLeave = useCallback(() => {
    if (active) setAsking(true);
    else leave();
  }, [active, leave]);

  const leaveDialog = (
    <ConfirmDialog
      open={asking}
      title="Thoát bài làm?"
      confirmLabel="Thoát"
      cancelLabel="Làm tiếp"
      danger
      onConfirm={leave}
      onCancel={() => setAsking(false)}
    >
      {detail && <p>{detail}</p>}
      <p>{LEAVE_MESSAGE}</p>
    </ConfirmDialog>
  );

  return { requestLeave, leaveDialog };
}
