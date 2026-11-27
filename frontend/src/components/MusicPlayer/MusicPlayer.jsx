import { useEffect, useRef, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { deleteMusic, uploadMusic } from '../../lib/api';
import { useCarolPlayer } from '../../lib/carolPlayer';
import { isQuietRoute } from '../../lib/quietRoutes';
import './MusicPlayer.css';

const signedIn = () => !!localStorage.getItem('access_token');
const mb = (bytes) => `${(bytes / (1 << 20)).toFixed(1)} MB`;

// MusicPlayer is the corner remote for the music box (lib/carolPlayer.js):
// on/off, previous/next and volume, plus "Nhạc của tôi", where a learner
// uploads their own songs (kept private to them) to play after the carols.
// It hides, and the music pauses, on pages that must stay quiet.
export default function MusicPlayer() {
  const player = useCarolPlayer();
  const { pathname } = useLocation();
  const quiet = isQuietRoute(pathname);
  const { setBlocked, refresh, on, libraryLoaded } = player;
  const [panelOpen, setPanelOpen] = useState(false);

  useEffect(() => {
    setBlocked(quiet);
  }, [quiet, setBlocked]);

  // The uploads are fetched once the player is in use (on, or the panel
  // opened) and someone is signed in, not on every page load.
  useEffect(() => {
    if ((on || panelOpen) && !libraryLoaded && signedIn() && pathname !== '/login' && pathname !== '/register') {
      refresh().catch(() => {});
    }
  }, [on, panelOpen, libraryLoaded, refresh, pathname]);

  if (quiet) return null;

  const { playing, title, index, count, volume, currentTrack } = player;
  const label = on ? 'Tắt nhạc Giáng sinh' : 'Bật nhạc Giáng sinh';

  return (
    <div className={`music-player ${on ? 'music-player-on' : ''}`} aria-label="Nhạc Giáng sinh">
      {panelOpen && <MyMusicPanel player={player} onClose={() => setPanelOpen(false)} />}

      <button
        type="button"
        role="switch"
        aria-checked={on}
        aria-label={label}
        title={label}
        className="music-switch"
        onClick={player.toggle}
      >
        <span className="music-switch-knob" aria-hidden="true">
          {on ? '♫' : '♪'}
        </span>
      </button>

      {!on && <span className="music-player-hint">Nhạc Noel</span>}

      {on && (
        <>
          <div className="music-now">
            <span className={`music-eq ${playing ? 'music-eq-playing' : ''}`} aria-hidden="true">
              <i />
              <i />
              <i />
            </span>
            <span className="music-title" title={title}>
              {currentTrack?.kind === 'user' && <span className="music-mine-dot" title="Nhạc của bạn" />}
              {title}
            </span>
            <span className="music-count">
              {index + 1}/{count}
              {!playing && ' · bấm bất kỳ đâu để phát'}
            </span>
          </div>
          <div className="music-buttons">
            <button type="button" className="music-btn" onClick={player.prev} aria-label="Bài trước" title="Bài trước">
              ⏮
            </button>
            <button type="button" className="music-btn" onClick={player.next} aria-label="Bài tiếp" title="Bài tiếp">
              ⏭
            </button>
          </div>
          <label className="music-volume" title={`Âm lượng ${Math.round(volume * 100)}%`}>
            <span aria-hidden="true">{volume === 0 ? '🔇' : volume < 0.5 ? '🔉' : '🔊'}</span>
            <input
              type="range"
              min="0"
              max="100"
              step="1"
              value={Math.round(volume * 100)}
              onChange={(e) => player.setVolume(Number(e.target.value) / 100)}
              aria-label="Âm lượng"
            />
          </label>
        </>
      )}

      {signedIn() && (
        <button
          type="button"
          className={`music-btn music-mine-btn ${panelOpen ? 'is-open' : ''}`}
          onClick={() => setPanelOpen((o) => !o)}
          aria-expanded={panelOpen}
          aria-label="Nhạc của tôi"
          title="Nhạc của tôi — tải bài hát của bạn lên"
        >
          ＋♫
        </button>
      )}
    </div>
  );
}

function MyMusicPanel({ player, onClose }) {
  const fileRef = useRef(null);
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState(null); // { name, done, total, fraction }
  const [error, setError] = useState('');
  const [limits, setLimits] = useState({ max_bytes: 15 << 20, max_count: 20 });

  const mine = player.list.map((t, i) => ({ ...t, index: i })).filter((t) => t.kind === 'user');

  async function handleFiles(files) {
    const list = Array.from(files || []);
    if (!list.length) return;
    setBusy(true);
    setError('');
    const failures = [];
    for (let i = 0; i < list.length; i++) {
      const file = list[i];
      if (file.size > limits.max_bytes) {
        failures.push(`${file.name}: quá ${mb(limits.max_bytes)}`);
        continue;
      }
      setProgress({ name: file.name, done: i, total: list.length, fraction: 0 });
      try {
        await uploadMusic(file, {
          onProgress: (f) => setProgress({ name: file.name, done: i, total: list.length, fraction: f }),
        });
      } catch (err) {
        failures.push(`${file.name}: ${err.message}`);
      }
    }
    try {
      const lib = await player.refresh();
      if (lib) setLimits({ max_bytes: lib.max_bytes, max_count: lib.max_count });
    } catch {
      // the list stays as it was
    }
    setProgress(null);
    setBusy(false);
    if (failures.length) setError(failures.join(' · '));
    if (fileRef.current) fileRef.current.value = '';
  }

  async function handleDelete(track) {
    setBusy(true);
    setError('');
    try {
      await deleteMusic(track.id);
      await player.refresh();
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="music-panel" role="dialog" aria-label="Nhạc của tôi">
      <div className="music-panel-head">
        <strong>Nhạc của tôi</strong>
        <button type="button" className="music-panel-close" onClick={onClose} aria-label="Đóng">
          ×
        </button>
      </div>
      <p className="music-panel-note">
        Chỉ mình bạn nghe được. Phát sau các bài Noel có sẵn. Tối đa {limits.max_count} bài, mỗi bài ≤ {mb(limits.max_bytes)}.
      </p>

      {mine.length === 0 && !progress && <p className="music-panel-empty">Chưa có bài nào — tải bài đầu tiên lên nhé.</p>}

      {mine.length > 0 && (
        <ol className="music-panel-list">
          {mine.map((t) => (
            <li key={t.key} className={t.index === player.index && player.on ? 'is-current' : ''}>
              <button
                type="button"
                className="music-panel-play"
                onClick={() => player.play(t.index)}
                title={`Phát "${t.title}"`}
                disabled={busy}
              >
                {t.index === player.index && player.playing ? '♫' : '▶'}
              </button>
              <span className="music-panel-title" title={t.title}>
                {t.title}
              </span>
              <button
                type="button"
                className="music-panel-delete"
                onClick={() => handleDelete(t)}
                aria-label={`Xoá ${t.title}`}
                title="Xoá bài này"
                disabled={busy}
              >
                ✕
              </button>
            </li>
          ))}
        </ol>
      )}

      {progress && (
        <div className="music-panel-progress" aria-live="polite">
          <span>
            Đang tải {progress.total > 1 ? `${progress.done + 1}/${progress.total} · ` : ''}
            {progress.name}
          </span>
          <span className="music-panel-bar">
            <i style={{ width: `${Math.round(progress.fraction * 100)}%` }} />
          </span>
        </div>
      )}

      {error && <p className="music-panel-error">{error}</p>}

      <input
        ref={fileRef}
        type="file"
        accept=".mp3,.m4a,.aac,.ogg,.oga,.opus,.wav,.webm,.flac,audio/*"
        multiple
        hidden
        onChange={(e) => handleFiles(e.target.files)}
      />
      <button
        type="button"
        className="music-panel-upload"
        onClick={() => fileRef.current?.click()}
        disabled={busy || mine.length >= limits.max_count}
      >
        {busy && progress ? 'Đang tải lên…' : '＋ Tải nhạc lên'}
      </button>
    </div>
  );
}
