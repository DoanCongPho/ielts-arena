import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  createCustomSpeaking,
  generateSpeakingParts,
  getTest,
  listTests,
  updateCustomSpeaking,
} from '../lib/api';
import { safeParse } from '../lib/safeParse';
import { seriesLabel } from '../lib/skillConfig';
import {
  EMPTY_PART1,
  EMPTY_PART2,
  EMPTY_PART3,
  MODES,
  MODE_LABELS,
  part1FromApi,
  part1ToApi,
  part2FromApi,
  part2ToApi,
  part3FromApi,
  part3ToApi,
  speakingSummary,
} from '../lib/speakingContent';
import { Part1Editor, Part2Editor, Part3Editor } from '../components/TestBuilder/SpeakingPartEditors';
import Button from '../components/ui/Button/Button';
import Card from '../components/ui/Card/Card';
import './CreateTestPage.css';

const PART_TITLES = {
  part1: 'Part 1 — Giới thiệu & phỏng vấn',
  part2: 'Part 2 — Cue card (nói 1–2 phút)',
  part3: 'Part 3 — Thảo luận',
};

// SpeakingComposerPage builds a user's own speaking test: a full test or
// one part, each part either taken from an official test or written here.
export default function SpeakingComposerPage() {
  const { testId } = useParams();
  const navigate = useNavigate();
  const editing = Boolean(testId);

  const [title, setTitle] = useState('');
  const [mode, setMode] = useState('part2');
  const [sources, setSources] = useState({ part1: 'bank', part2: 'custom', part3: 'custom' });
  const [bankIds, setBankIds] = useState({ part1: '', part2: '', part3: '' });
  const [part1, setPart1] = useState(EMPTY_PART1);
  const [part2, setPart2] = useState(EMPTY_PART2);
  const [part3, setPart3] = useState(EMPTY_PART3);
  const [bank, setBank] = useState([]);

  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(null);

  useEffect(() => {
    listTests('speaking', 1)
      .then((data) => setBank((data.data || []).map((t) => ({ ...t, content: safeParse(t.content_data) }))))
      .catch(() => setBank([]));
  }, []);

  // Editing loads the saved content with every part as the user's own:
  // where a part originally came from isn't kept.
  useEffect(() => {
    if (!testId) return;
    getTest(testId)
      .then((t) => {
        const c = safeParse(t.content_data) || {};
        setTitle(c.title || '');
        setMode(t.task_type);
        setSources({ part1: 'custom', part2: 'custom', part3: 'custom' });
        if (c.part1) setPart1(part1FromApi(c.part1));
        if (c.part2) setPart2(part2FromApi(c.part2));
        if (c.part3) setPart3(part3FromApi(c.part3));
      })
      .catch((err) => setError(err.message));
  }, [testId]);

  const parts = MODES.find((m) => m.key === mode).parts;

  function body() {
    const out = { title };
    const custom = { part1: part1ToApi(part1), part2: part2ToApi(part2), part3: part3ToApi(part3) };
    for (const p of parts) {
      out[p] = sources[p] === 'bank' ? { bank_test_id: Number(bankIds[p]) } : { custom: custom[p] };
    }
    return out;
  }

  async function handleGenerate() {
    setBusy('generate');
    setError('');
    try {
      const drafted = await generateSpeakingParts(part2ToApi(part2), {
        part1: sources.part1 === 'custom',
        part3: sources.part3 === 'custom',
      });
      if (drafted.part1) setPart1(part1FromApi(drafted.part1));
      if (drafted.part3) setPart3(part3FromApi(drafted.part3));
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy('');
    }
  }

  async function handleSave(e) {
    e.preventDefault();
    for (const p of parts) {
      if (sources[p] === 'bank' && !bankIds[p]) {
        setError(`Chọn đề có sẵn cho ${MODE_LABELS[p]}.`);
        return;
      }
    }
    setBusy('save');
    setError('');
    try {
      const res = editing ? await updateCustomSpeaking(testId, body()) : await createCustomSpeaking(body());
      if (res.warnings?.length) setSaved(res);
      else navigate(`/practice/speaking/${res.test.id}`);
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy('');
    }
  }

  const canGenerate = mode === 'full' && sources.part2 === 'custom' && (sources.part1 === 'custom' || sources.part3 === 'custom');

  return (
    <div className="tb-page">
      <header className="tb-header">
        <Button variant="secondary" onClick={() => navigate('/practice/speaking')}>
          ← Speaking
        </Button>
        <h1 className="text-h1">{editing ? 'Sửa đề của tôi' : 'Tạo đề Speaking của tôi'}</h1>
      </header>

      <form className="tb-form" onSubmit={handleSave}>
        <Card className="tb-meta-card">
          <label className="tb-field tb-field-inline">
            <span>Tên đề</span>
            <input className="tb-input" value={title} maxLength={100} onChange={(e) => setTitle(e.target.value)} placeholder="Đề luyện chủ đề sách" />
          </label>
          <label className="tb-field tb-field-inline">
            <span>Luyện</span>
            <select className="tb-select" value={mode} onChange={(e) => setMode(e.target.value)}>
              {MODES.map((m) => <option key={m.key} value={m.key}>{m.label}</option>)}
            </select>
          </label>
          {mode !== 'full' && (
            <p className="tb-hint">Luyện một phần được chấm theo cùng tiêu chí, nhưng band chỉ mang tính tham khảo — IELTS không chấm riêng từng phần.</p>
          )}
        </Card>

        {parts.map((p) => (
          <Card key={p} className="tb-group-card">
            <p className="tb-card-eyebrow">{PART_TITLES[p]}</p>
            <div className="tb-field tb-field-inline-group">
              <label className="tb-field tb-field-checkbox">
                <input type="radio" checked={sources[p] === 'bank'} onChange={() => setSources({ ...sources, [p]: 'bank' })} />
                <span>Lấy từ đề có sẵn</span>
              </label>
              <label className="tb-field tb-field-checkbox">
                <input type="radio" checked={sources[p] === 'custom'} onChange={() => setSources({ ...sources, [p]: 'custom' })} />
                <span>Tự viết</span>
              </label>
            </div>

            {sources[p] === 'bank' ? (
              <select className="tb-select" value={bankIds[p]} onChange={(e) => setBankIds({ ...bankIds, [p]: e.target.value })}>
                <option value="">— Chọn đề —</option>
                {bank.filter((t) => t.content?.[p]).map((t) => (
                  <option key={t.id} value={t.id}>
                    {[seriesLabel(t), t.test_number && `Test ${t.test_number}`, speakingSummary(t.content)].filter(Boolean).join(' · ')}
                  </option>
                ))}
              </select>
            ) : p === 'part1' ? (
              <Part1Editor value={part1} onChange={setPart1} />
            ) : p === 'part2' ? (
              <Part2Editor value={part2} onChange={setPart2} />
            ) : (
              <Part3Editor value={part3} onChange={setPart3} needsTheme={mode === 'part3'} />
            )}

            {p === 'part2' && canGenerate && (
              <Button type="button" variant="secondary" onClick={handleGenerate} disabled={busy !== ''}>
                {busy === 'generate' ? 'Đang soạn...' : `Tự tạo ${[sources.part1 === 'custom' && 'Part 1', sources.part3 === 'custom' && 'Part 3'].filter(Boolean).join(' và ')} từ cue card này`}
              </Button>
            )}
          </Card>
        ))}

        {error && <p className="tb-error">{error}</p>}
        {saved && (
          <div className="tb-success">
            <p>Đã lưu. Một vài gợi ý để đề giống đề thật hơn:</p>
            <ul>{saved.warnings.map((w) => <li key={w}>{w}</li>)}</ul>
            <Button type="button" variant="primary" onClick={() => navigate(`/practice/speaking/${saved.test.id}`)}>
              Làm bài ngay
            </Button>
          </div>
        )}

        <Button type="submit" variant="primary" className="tb-submit-btn" disabled={busy !== ''}>
          {busy === 'save' ? 'Đang lưu...' : editing ? 'Lưu thay đổi' : 'Lưu và làm bài'}
        </Button>
      </form>
    </div>
  );
}
