import { useSyncExternalStore } from 'react';
import { isServerWaking, onServerWakingChange } from '../../lib/api';
import './ServerWakeBanner.css';

// ServerWakeBanner tells the learner the API is booting (Render's free
// plan sleeps when idle) while apiFetch waits for it, instead of leaving
// them looking at an error.
export default function ServerWakeBanner() {
  const waking = useSyncExternalStore(onServerWakingChange, isServerWaking, () => false);
  if (!waking) return null;
  return (
    <div className="server-wake-banner" role="status" aria-live="polite">
      <span className="server-wake-spinner" aria-hidden="true" />
      <span>
        <strong>Máy chủ đang khởi động, đợi chút…</strong>
        <small>Lần đầu vào sau một lúc không dùng có thể mất tới 1 phút.</small>
      </span>
    </div>
  );
}
