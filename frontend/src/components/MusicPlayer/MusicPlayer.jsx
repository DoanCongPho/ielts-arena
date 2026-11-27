import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { useCarolPlayer } from '../../lib/carolPlayer';
import { isQuietRoute } from '../../lib/quietRoutes';
import './MusicPlayer.css';

// MusicPlayer is the corner remote for the Christmas music box
// (lib/carolPlayer.js): on/off, previous/next carol and volume. It hides,
// and the music pauses, on pages that must stay quiet.
export default function MusicPlayer() {
  const player = useCarolPlayer();
  const { pathname } = useLocation();
  const quiet = isQuietRoute(pathname);
  const { setBlocked } = player;

  useEffect(() => {
    setBlocked(quiet);
  }, [quiet, setBlocked]);

  if (quiet) return null;

  const { on, playing, title, track, count, volume } = player;
  const label = on ? 'Tắt nhạc Giáng sinh' : 'Bật nhạc Giáng sinh';

  return (
    <div className={`music-player ${on ? 'music-player-on' : ''}`} aria-label="Nhạc Giáng sinh">
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
              {title}
            </span>
            <span className="music-count">
              {track + 1}/{count}
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
    </div>
  );
}
