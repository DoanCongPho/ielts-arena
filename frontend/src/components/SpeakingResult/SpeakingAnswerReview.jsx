import { useEffect, useMemo, useRef, useState } from 'react';

// Word scores (0-100) from the pronunciation service: below FAIR the word
// is marked as mispronounced, between FAIR and GOOD as unclear.
const GOOD = 80;
const FAIR = 60;

const norm = (s) => (s || '').toLowerCase().replace(/[^a-z0-9']/g, '');

function wordClass(score) {
  if (score == null) return '';
  if (score < FAIR) return 'sa-word-bad';
  if (score < GOOD) return 'sa-word-fair';
  return 'sa-word-good';
}

// saidLine is what the candidate said in place of each expected sound.
// It is built from the per-sound alignment rather than the raw decoding of
// the word's time span, which for short words ("a", "to") also picks up
// the neighbouring words' sounds.
function saidLine(phonemes) {
  return phonemes.map((p) => (p.score >= GOOD ? p.expected : p.heard || '∅')).join(' ') || '—';
}

// placeCorrections finds each correction's original text in the answer's
// recognised words, so it can be struck through in place. Corrections that
// can't be placed (the text and the word list differ slightly) are
// returned separately and listed under the answer instead.
function placeCorrections(words, corrections) {
  const tokens = words.map((w) => norm(w.word));
  const taken = new Array(tokens.length).fill(false);
  const placed = new Map(); // start index -> {end, correction}
  const unplaced = [];
  for (const c of corrections) {
    const target = c.original.split(/\s+/).map(norm).filter(Boolean);
    let found = -1;
    for (let i = 0; i + target.length <= tokens.length && found < 0; i++) {
      if (target.every((t, k) => tokens[i + k] === t && !taken[i + k])) found = i;
    }
    if (found < 0 || !target.length) {
      unplaced.push(c);
      continue;
    }
    for (let k = 0; k < target.length; k++) taken[found + k] = true;
    placed.set(found, { end: found + target.length - 1, correction: c });
  }
  return { placed, unplaced };
}

// SpeakingAnswerReview shows one answer: the recording, the transcript with
// each word coloured by how clearly it was pronounced, grammar and word
// choice errors struck through with their fix, and — for a clicked word —
// what the candidate said next to the expected pronunciation.
export default function SpeakingAnswerReview({ answer, pronWords, corrections, recordingUrl }) {
  const audioRef = useRef(null);
  const stopAt = useRef(null);
  const [selected, setSelected] = useState(null);

  const byIndex = useMemo(() => new Map((pronWords || []).map((w) => [w.index, w])), [pronWords]);
  const words = useMemo(() => answer.words || [], [answer.words]);
  const { placed, unplaced } = useMemo(() => placeCorrections(words, corrections || []), [words, corrections]);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return undefined;
    const onTime = () => {
      if (stopAt.current != null && audio.currentTime >= stopAt.current) {
        audio.pause();
        stopAt.current = null;
      }
    };
    audio.addEventListener('timeupdate', onTime);
    return () => audio.removeEventListener('timeupdate', onTime);
  }, [recordingUrl]);

  // playSpan plays the candidate's own recording of one word, with a
  // little context either side so it doesn't sound clipped.
  function playSpan(start, end) {
    const audio = audioRef.current;
    if (!audio) return;
    audio.currentTime = Math.max(0, start - 0.15);
    stopAt.current = end + 0.15;
    audio.play();
  }

  function playReference(text) {
    if (typeof speechSynthesis === 'undefined') return;
    speechSynthesis.cancel();
    const u = new SpeechSynthesisUtterance(text);
    u.lang = 'en-GB';
    u.rate = 0.85;
    speechSynthesis.speak(u);
  }

  const elements = [];
  for (let i = 0; i < words.length; i++) {
    const fix = placed.get(i);
    if (fix) {
      const said = words.slice(i, fix.end + 1).map((w) => w.word.trim()).join(' ');
      elements.push(
        <span key={`c${i}`} className={`sa-fix sa-fix-${fix.correction.type}`} title={fix.correction.explanation}>
          <del>{said}</del> <ins>{fix.correction.correction}</ins>
        </span>,
        ' ',
      );
      i = fix.end;
      continue;
    }
    const w = words[i];
    const pw = byIndex.get(i);
    elements.push(
      <button
        key={i}
        type="button"
        className={`sa-word ${wordClass(pw?.score)} ${selected === i ? 'sa-word-selected' : ''}`}
        onClick={() => setSelected(selected === i ? null : i)}
        title={pw ? `Phát âm: ${Math.round(pw.score)}/100` : undefined}
      >
        {w.word.trim()}
      </button>,
      ' ',
    );
  }

  const sel = selected != null ? { word: words[selected], pron: byIndex.get(selected) } : null;

  return (
    <div className="sa">
      {recordingUrl && <audio ref={audioRef} className="speaking-result-audio" controls preload="metadata" src={recordingUrl} />}

      {words.length ? <p className="sa-text">{elements}</p> : <p className="sa-text">(không có lời nói)</p>}

      {sel && (
        <div className="sa-panel">
          <div className="sa-panel-head">
            <strong>{sel.word.word.trim()}</strong>
            {sel.pron && <span className={`sa-score ${wordClass(sel.pron.score)}`}>{Math.round(sel.pron.score)}/100</span>}
            <button type="button" className="sa-play" onClick={() => playSpan(sel.word.start, sel.word.end)} disabled={!recordingUrl}>
              ▶ Bạn đọc
            </button>
            <button type="button" className="sa-play" onClick={() => playReference(sel.word.word.trim())}>
              ▶ Đọc chuẩn
            </button>
          </div>
          {sel.pron ? (
            <>
              <div className="sa-ipa">
                <span className="sa-ipa-label">Chuẩn</span>
                <span className="sa-ipa-value">/{sel.pron.expected}/</span>
              </div>
              <div className="sa-ipa">
                <span className="sa-ipa-label">Bạn đọc</span>
                <span className="sa-ipa-value">/{saidLine(sel.pron.phonemes)}/</span>
              </div>
              <div className="sa-phones">
                {sel.pron.phonemes.map((p, k) => (
                  <span key={k} className={`sa-phone ${wordClass(p.score)}`} title={`${Math.round(p.score)}/100`}>
                    <span className="sa-phone-exp">{p.expected}</span>
                    <span className="sa-phone-heard">
                      {p.score >= GOOD ? '✓' : p.heard ? p.heard : 'mất'}
                    </span>
                  </span>
                ))}
              </div>
              <p className="sa-hint">
                Mỗi ô là một âm: dòng trên là âm chuẩn, dòng dưới là âm máy nghe được từ giọng bạn. Phiên âm được so với cả giọng Anh lẫn Mỹ.
              </p>
            </>
          ) : (
            <p className="sa-hint">Từ này chưa được chấm phát âm.</p>
          )}
        </div>
      )}

      {(corrections?.length > 0) && (
        <ul className="sa-corrections">
          {corrections.map((c, k) => (
            <li key={k} className={unplaced.includes(c) ? 'sa-corr-unplaced' : ''}>
              <span className={`sa-corr-type sa-corr-type-${c.type}`}>{c.type === 'grammar' ? 'Ngữ pháp' : 'Từ vựng'}</span>
              <del>{c.original}</del> → <ins>{c.correction}</ins>
              {c.explanation && <span className="sa-corr-why"> — {c.explanation}</span>}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
