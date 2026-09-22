import { useCallback, useEffect, useRef, useState } from 'react';

// Reading and writing attempts get a fixed time allowance.
export const ATTEMPT_SECONDS = 60 * 60;

// A listening attempt lasts as long as its recording plus this, to check
// answers once the last part has played.
export const LISTENING_CHECK_SECONDS = 2 * 60;

const LEAVE_MESSAGE = 'Bài làm chưa nộp sẽ bị mất. Bạn có chắc muốn rời trang?';

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

// useLeaveGuard stops an in-progress attempt from being left by accident —
// a trackpad swipe-back, the browser Back button, a reload or closing the
// tab. While `active`:
//   - horizontal overscroll is disabled, so a two-finger swipe doesn't start
//     the browser's back gesture in the first place;
//   - an extra history entry for the current page is pushed, so Back lands
//     on it instead of leaving, and the user is asked to confirm; confirming
//     goes back for real, cancelling re-arms the trap;
//   - reload/close shows the browser's native "leave site?" prompt.
// It returns confirmLeave() for in-app exits (e.g. the page's back arrow):
// true if it's fine to navigate away.
export function useLeaveGuard(active) {
  const leaving = useRef(false);

  useEffect(() => {
    if (!active) return;

    const root = document.documentElement;
    const prevOverscroll = root.style.overscrollBehaviorX;
    root.style.overscrollBehaviorX = 'none';

    // Re-use the router's own history state so React Router treats the
    // extra entry as the same location.
    window.history.pushState(window.history.state, '', window.location.href);

    function onPopState() {
      if (leaving.current) return;
      if (window.confirm(LEAVE_MESSAGE)) {
        leaving.current = true;
        window.history.back();
      } else {
        window.history.pushState(window.history.state, '', window.location.href);
      }
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

  return useCallback(() => !active || window.confirm(LEAVE_MESSAGE), [active]);
}
