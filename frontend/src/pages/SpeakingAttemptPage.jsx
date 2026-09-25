import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  getScore,
  getSpeakingScript,
  requestUploadSlots,
  submitAnswer,
  uploadRecording,
  waitForGrading,
} from '../lib/api';
import { openMicrophone, recordingFormat, speakLine, startRecording } from '../lib/speakingAudio';
import { useLeaveGuard } from '../lib/useAttemptSession';
import SpeakingResult from '../components/SpeakingResult/SpeakingResult';
import Button from '../components/ui/Button/Button';
import './WritingAttemptPage.css';
import './SpeakingAttemptPage.css';

// Part 1 and 3 limits are soft, as in the real test: the timer shows the
// usual length and the examiner moves on this much later. The Part 2 talk
// stops at exactly two minutes.
const SOFT_LIMIT_GRACE = 30;
// Answers can't be ended before this, so a stray click doesn't skip one.
const MIN_ANSWER_SECONDS = 2;
// Speaking grades transcribe every answer and wait on the pronunciation
// service, which may be waking up — minutes, not seconds.
const GRADING_TIMEOUT_MS = 20 * 60 * 1000;

const MODE_TITLES = {
  full: 'Speaking — Full test',
  part1: 'Speaking — Part 1',
  part2: 'Speaking — Part 2',
  part3: 'Speaking — Part 3',
};

export default function SpeakingAttemptPage() {
  const { testId } = useParams();
  const navigate = useNavigate();

  const [script, setScript] = useState(null);
  const [error, setError] = useState('');
  // ready → running → submitting → grading → done | failed
  const [phase, setPhase] = useState('loading');
  const [index, setIndex] = useState(0);
  // What the current line is doing: speak (examiner talking), prep (Part 2
  // preparation), record (candidate answering).
  const [stage, setStage] = useState('speak');
  const [clock, setClock] = useState({ deadline: 0, hard: 0, started: 0 });
  const [now, setNow] = useState(Date.now());
  const [showText, setShowText] = useState(false);
  const [notes, setNotes] = useState('');
  const [score, setScore] = useState(null);
  const [submissionId, setSubmissionId] = useState(null);

  const skipRef = useRef(null);
  const abortRef = useRef(null);
  const streamRef = useRef(null);

  useEffect(() => {
    getSpeakingScript(testId)
      .then((s) => {
        setScript(s);
        setPhase('ready');
      })
      .catch((err) => setError(err.message));
  }, [testId]);

  // Stop the examiner and release the microphone when leaving the page.
  useEffect(() => () => {
    abortRef.current?.abort();
    streamRef.current?.getTracks().forEach((t) => t.stop());
  }, []);

  useEffect(() => {
    if (stage === 'speak' || phase !== 'running') return undefined;
    const id = setInterval(() => setNow(Date.now()), 250);
    return () => clearInterval(id);
  }, [stage, phase]);

  const confirmLeave = useLeaveGuard(phase === 'running' || phase === 'submitting');

  // waitFor resolves after `seconds`, or earlier when the candidate clicks
  // the stage's button (skipRef). `display` is the target shown on the
  // timer; `seconds` the hard stop.
  function waitFor(seconds, display = seconds) {
    const started = Date.now();
    setClock({ started, deadline: started + display * 1000, hard: started + seconds * 1000 });
    setNow(started);
    return new Promise((resolve) => {
      const timer = setTimeout(done, seconds * 1000);
      function done() {
        clearTimeout(timer);
        skipRef.current = null;
        resolve();
      }
      skipRef.current = done;
    });
  }

  async function begin() {
    setError('');
    const format = recordingFormat();
    if (!format) {
      setError('Trình duyệt này không hỗ trợ ghi âm. Hãy dùng Chrome, Edge, Firefox hoặc Safari bản mới.');
      return;
    }
    let stream;
    try {
      stream = await openMicrophone();
    } catch {
      setError('Không truy cập được micro. Hãy cho phép trang dùng micro rồi thử lại.');
      return;
    }
    streamRef.current = stream;
    const answerable = script.lines.filter((l) => l.question_id);
    let slots;
    try {
      slots = await requestUploadSlots(answerable.length, format.ext);
    } catch (err) {
      setError(err.message);
      return;
    }
    setPhase('running');
    runExam(stream, format, slots);
  }

  async function runExam(stream, format, slots) {
    const abort = new AbortController();
    abortRef.current = abort;
    const answers = [];
    const uploads = [];
    let slot = 0;

    try {
      for (let i = 0; i < script.lines.length; i++) {
        const line = script.lines[i];
        setIndex(i);
        setStage('speak');
        await speakLine(line, abort.signal);

        if (line.kind === 'cue_card') {
          setStage('prep');
          await waitFor(line.seconds);
          continue;
        }
        if (!line.question_id) continue;

        setStage('record');
        const stop = startRecording(stream, format);
        const hard = line.kind === 'long_turn' ? line.seconds : line.seconds + SOFT_LIMIT_GRACE;
        await waitFor(hard, line.seconds);
        const { blob, durationSec } = await stop();

        const { key, upload_url: uploadUrl } = slots[slot++];
        answers.push({ question_id: line.question_id, audio_key: key, duration_sec: Math.round(durationSec * 10) / 10 });
        // Upload while the test goes on; failures are collected, not thrown,
        // until everything is awaited at the end.
        uploads.push(uploadRecording(uploadUrl, blob).then(() => null, (err) => err));
      }
    } catch (err) {
      if (abort.signal.aborted) return;
      setError(err.message);
      return;
    } finally {
      stream.getTracks().forEach((t) => t.stop());
    }

    setPhase('submitting');
    try {
      const failed = (await Promise.all(uploads)).find(Boolean);
      if (failed) throw failed;
      const queued = await submitAnswer(Number(testId), { answers });
      setSubmissionId(queued.id);
      setPhase('grading');
      const settled = await waitForGrading(queued.id, { intervalMs: 4000, timeoutMs: GRADING_TIMEOUT_MS });
      if (settled.status === 'graded') {
        setScore(await getScore(settled.id));
        setPhase('done');
      } else {
        setPhase('failed');
      }
    } catch (err) {
      setError(err.message);
    }
  }

  if (!script) {
    return (
      <div className="attempt-page">
        <p className={`practice-status ${error ? 'practice-error' : ''}`}>{error || 'Đang tải đề...'}</p>
      </div>
    );
  }

  const line = script.lines[index];
  const cueCard = script.lines.find((l) => l.kind === 'cue_card');
  const inPart2Talk = phase === 'running' && (stage === 'prep' || line.kind === 'long_turn') && line.part === 2;
  const answerable = script.lines.filter((l) => l.question_id);
  const answeredSoFar = script.lines.slice(0, index).filter((l) => l.question_id).length;
  const elapsed = Math.max(0, Math.floor((now - clock.started) / 1000));
  const remaining = Math.max(0, Math.ceil((clock.deadline - now) / 1000));
  const overTarget = stage === 'record' && now > clock.deadline;

  return (
    <div className="attempt-page attempt-page-focus">
      <header className="attempt-header">
        <button
          type="button"
          className="attempt-back"
          onClick={() => confirmLeave() && navigate('/practice/speaking')}
          aria-label="Về danh sách đề"
          title="Về danh sách đề"
        >
          ←
        </button>
        <span className="speaking-title">{MODE_TITLES[script.mode] || 'Speaking'}</span>
        {phase === 'running' && (
          <span className="attempt-timer">
            Part {line.part} · {Math.min(answeredSoFar + 1, answerable.length)}/{answerable.length}
          </span>
        )}
      </header>

      {phase === 'ready' && (
        <div className="speaking-stage">
          <h2 className="speaking-stage-title">Trước khi bắt đầu</h2>
          <ul className="speaking-rules">
            <li>Giám khảo sẽ đọc từng câu hỏi. Khi giám khảo nói xong, máy tự ghi âm câu trả lời của bạn.</li>
            <li>Mỗi câu chỉ trả lời một lần, không ghi âm lại — giống bài thi thật.</li>
            <li>Part 2: bạn có 1 phút chuẩn bị (được ghi chú), rồi nói tối đa 2 phút.</li>
            <li>Ngồi ở nơi yên tĩnh và nói rõ ràng. Bài được chấm theo 4 tiêu chí của IELTS.</li>
          </ul>
          {error && <p className="practice-status practice-error">{error}</p>}
          <Button variant="primary" onClick={begin}>Bắt đầu — cho phép micro</Button>
        </div>
      )}

      {phase === 'running' && (
        <div className={`speaking-stage ${inPart2Talk ? 'speaking-stage-card' : ''}`}>
          {inPart2Talk && cueCard ? (
            <CueCard text={cueCard.text} />
          ) : (
            <div className="speaking-examiner">
              <div className={`speaking-examiner-avatar ${stage === 'speak' ? 'speaking-examiner-talking' : ''}`} aria-hidden>
                🎙️
              </div>
              <p className="speaking-examiner-label text-label">Giám khảo</p>
              {showText ? (
                <p className="speaking-line-text">{line.text}</p>
              ) : (
                <p className="speaking-line-hidden">Hãy nghe câu hỏi</p>
              )}
              <button type="button" className="speaking-toggle" onClick={() => setShowText((v) => !v)}>
                {showText ? 'Ẩn câu hỏi' : 'Hiện câu hỏi'}
              </button>
            </div>
          )}

          {stage === 'prep' && (
            <div className="speaking-controls">
              <textarea
                className="speaking-notes"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Ghi chú ý chính (không được chấm)..."
              />
              <span className="speaking-clock">Chuẩn bị: {formatTime(remaining)}</span>
              <Button variant="secondary" onClick={() => skipRef.current?.()}>Bắt đầu nói ngay</Button>
            </div>
          )}

          {stage === 'record' && (
            <div className="speaking-controls">
              <span className="speaking-rec"><span className="speaking-rec-dot" /> Đang ghi âm</span>
              <span className={`speaking-clock ${overTarget ? 'attempt-timer-low' : ''}`}>
                {line.kind === 'long_turn' ? `Còn ${formatTime(remaining)}` : formatTime(elapsed)}
              </span>
              <Button
                variant="primary"
                disabled={elapsed < MIN_ANSWER_SECONDS}
                onClick={() => skipRef.current?.()}
              >
                Xong
              </Button>
            </div>
          )}

          {stage === 'prep' && (
            <p className="practice-status">Bạn có một phút để chuẩn bị. Giám khảo sẽ báo khi đến lúc nói.</p>
          )}
          {error && <p className="practice-status practice-error">{error}</p>}
        </div>
      )}

      {(phase === 'submitting' || phase === 'grading') && (
        <div className="speaking-stage">
          <h2 className="speaking-stage-title">{phase === 'submitting' ? 'Đang nộp bài...' : 'Đang chấm điểm...'}</h2>
          <p className="practice-status">
            Bài nói được chép lại, đo độ trôi chảy và phát âm, rồi chấm theo 4 tiêu chí của IELTS. Việc này có thể mất vài phút —
            bạn có thể rời trang và xem kết quả trong mục Lịch sử làm bài.
          </p>
          {error && <p className="practice-status practice-error">{error}</p>}
          {submissionId && (
            <Button variant="secondary" onClick={() => navigate('/submissions')}>Xem sau trong Lịch sử</Button>
          )}
        </div>
      )}

      {phase === 'failed' && (
        <div className="speaking-stage">
          <p className="practice-status practice-error">
            Bài đã được lưu nhưng chấm điểm thất bại. Xem lại trong mục Lịch sử làm bài.
          </p>
        </div>
      )}

      {phase === 'done' && score && <SpeakingResult score={score} submissionId={submissionId} />}
    </div>
  );
}

function CueCard({ text }) {
  const [topic, , ...rest] = text.split('\n');
  const bullets = rest.filter((l) => l.startsWith('- ')).map((l) => l.slice(2));
  const explain = rest.find((l) => !l.startsWith('- '));
  return (
    <div className="speaking-cue-card">
      <p className="speaking-cue-topic">{topic}</p>
      <p className="speaking-cue-say">You should say:</p>
      <ul>
        {bullets.map((b) => <li key={b}>{b}</li>)}
      </ul>
      {explain && <p className="speaking-cue-explain">{explain}</p>}
    </div>
  );
}

function formatTime(totalSeconds) {
  const m = Math.floor(totalSeconds / 60).toString().padStart(2, '0');
  const s = (totalSeconds % 60).toString().padStart(2, '0');
  return `${m}:${s}`;
}
