import { useEffect, useRef, useState } from 'react';
import { playlist, sectionAt, totalSeconds } from '../../lib/listeningAudio';
import './ListeningExamPlayer.css';

// ListeningExamPlayer plays a listening test once, straight through, like
// the real exam: no visible player, no pause or seeking. Sections with
// their own recording (section.audio_url) play back to back; a test with
// one shared recording (content.audio_url) plays that file. The <audio>
// stays mounted for the whole attempt, so switching section tabs or
// jumping between questions never touches playback.
//
// Before `started` it shows a start button (browsers only allow sound
// after a click); onStart is called from that click.
export default function ListeningExamPlayer({ content, started, onStart }) {
  const audioRef = useRef(null);
  const tracks = playlist(content);
  const [track, setTrack] = useState(0);
  const [position, setPosition] = useState(0);
  const [finished, setFinished] = useState(false);
  const [failed, setFailed] = useState(false);
  const [volume, setVolume] = useState(1);

  const total = totalSeconds(content);
  const before = tracks.slice(0, track).reduce((sum, t) => sum + t.length, 0);
  const elapsed = Math.min(total, before + position);
  const playingSection = tracks.length > 1 ? track : sectionAt(content, position);

  useEffect(() => {
    if (audioRef.current) audioRef.current.volume = volume;
  }, [volume]);

  function start() {
    onStart();
    audioRef.current?.play()?.catch(() => setFailed(true));
  }

  function handleEnded() {
    if (track + 1 < tracks.length) {
      setTrack(track + 1);
      setPosition(0);
    } else {
      setFinished(true);
    }
  }

  // Moving to the next track swaps src; start it as soon as it can play.
  function handleCanPlay() {
    if (started && !finished && audioRef.current?.paused) {
      audioRef.current.play()?.catch(() => setFailed(true));
    }
  }

  return (
    <div className="exam-player">
      <audio
        ref={audioRef}
        src={tracks[track]?.src}
        preload="auto"
        onTimeUpdate={(e) => setPosition(e.currentTarget.currentTime)}
        onEnded={handleEnded}
        onCanPlay={handleCanPlay}
        onError={() => setFailed(true)}
      />

      {!started ? (
        <div className="exam-player-intro">
          <p className="exam-player-note">
            Bài nghe gồm {content?.sections?.length || 0} phần, phát liên tục <strong>một lần duy nhất</strong> (khoảng{' '}
            {Math.round(total / 60)} phút) — không tạm dừng hay tua lại được. Sau khi phát hết còn 2 phút để kiểm tra đáp án.
          </p>
          <button type="button" className="exam-player-start" onClick={start}>
            ▶ Bắt đầu nghe
          </button>
        </div>
      ) : (
        <div className="exam-player-status">
          <span className={`exam-player-dot ${finished ? 'exam-player-dot-done' : ''}`} aria-hidden="true" />
          <span className="exam-player-label">
            {finished ? 'Đã phát xong — kiểm tra lại đáp án' : `Đang phát Part ${playingSection + 1}`}
          </span>
          <span className="exam-player-time text-data-sm">
            {formatClock(elapsed)} / {formatClock(total)}
          </span>
          <label className="exam-player-volume">
            <span aria-hidden="true">🔊</span>
            <input
              type="range"
              min="0"
              max="1"
              step="0.05"
              value={volume}
              aria-label="Âm lượng"
              onChange={(e) => setVolume(Number(e.target.value))}
            />
          </label>
        </div>
      )}
      {failed && <p className="practice-status practice-error">Không phát được audio — tải lại trang để thử lại.</p>}
      <div className="exam-player-progress" aria-hidden="true">
        <div style={{ width: `${total ? (elapsed / total) * 100 : 0}%` }} />
      </div>
    </div>
  );
}

function formatClock(totalSec) {
  const s = Math.floor(totalSec);
  return `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`;
}
