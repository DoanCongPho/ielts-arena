import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { createTest } from '../lib/api';
import { buildTestPayload } from '../lib/buildTestPayload';
import WritingBuilder from '../components/TestBuilder/WritingBuilder';
import PassagesEditor from '../components/TestBuilder/PassagesEditor';
import SectionsEditor from '../components/TestBuilder/SectionsEditor';
import Button from '../components/ui/Button/Button';
import Card from '../components/ui/Card/Card';
import './CreateTestPage.css';

export default function CreateTestPage() {
  const navigate = useNavigate();

  const [skill, setSkill] = useState('reading');
  const [taskType, setTaskType] = useState('');
  const [source, setSource] = useState('manual');
  const [isCurrent, setIsCurrent] = useState(true);
  const [xpGain, setXpGain] = useState(50);
  const [thumbnailUrl, setThumbnailUrl] = useState('');

  const [writingContent, setWritingContent] = useState({ prompt: '', image_url: '' });
  const [passages, setPassages] = useState([{ title: '', paragraphs: [{ label: '', text: '' }], question_groups: [] }]);
  const [listeningAudioUrl, setListeningAudioUrl] = useState('');
  const [sections, setSections] = useState([{ title: '', section_start_time: 0, section_end_time: 0, question_groups: [] }]);

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(null);

  async function handleSubmit(e) {
    e.preventDefault();
    setError('');
    setSuccess(null);

    if (!taskType.trim()) {
      setError('task_type là bắt buộc.');
      return;
    }

    const payload = buildTestPayload({
      skill,
      taskType,
      source,
      isCurrent,
      xpGain,
      thumbnailUrl,
      writingContent,
      passages,
      listeningAudioUrl,
      sections,
    });

    setSubmitting(true);
    try {
      const created = await createTest(payload);
      setSuccess(created);
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="tb-page">
      <header className="tb-header">
        <Button variant="secondary" onClick={() => navigate('/dashboard')}>
          ← Trang chủ
        </Button>
        <h1 className="text-h1">Tạo đề thi</h1>
      </header>

      <form className="tb-form" onSubmit={handleSubmit}>
        <Card className="tb-meta-card">
          <label className="tb-field tb-field-inline">
            <span>Kỹ năng</span>
            <select className="tb-select" value={skill} onChange={(e) => setSkill(e.target.value)}>
              <option value="reading">Reading</option>
              <option value="listening">Listening</option>
              <option value="writing">Writing</option>
            </select>
          </label>

          <label className="tb-field tb-field-inline">
            <span>Task type (vd: test1, task1, section2...)</span>
            <input className="tb-input" value={taskType} onChange={(e) => setTaskType(e.target.value)} required />
          </label>

          <label className="tb-field tb-field-inline">
            <span>Nguồn (source)</span>
            <input className="tb-input" value={source} onChange={(e) => setSource(e.target.value)} />
          </label>

          <label className="tb-field tb-field-inline">
            <span>XP thưởng (xp_gain)</span>
            <input
              type="number"
              min={0}
              className="tb-input tb-input-narrow"
              value={xpGain}
              onChange={(e) => setXpGain(Number(e.target.value))}
            />
          </label>

          <label className="tb-field tb-field-inline">
            <span>Thumbnail URL (tuỳ chọn)</span>
            <input className="tb-input" value={thumbnailUrl} onChange={(e) => setThumbnailUrl(e.target.value)} />
          </label>

          <label className="tb-field tb-field-checkbox">
            <input type="checkbox" checked={isCurrent} onChange={(e) => setIsCurrent(e.target.checked)} />
            <span>Hiển thị đề này (is_current)</span>
          </label>
        </Card>

        {skill === 'writing' && <WritingBuilder content={writingContent} onChange={setWritingContent} />}

        {skill === 'reading' && <PassagesEditor passages={passages} onChange={setPassages} />}

        {skill === 'listening' && (
          <>
            <label className="tb-field">
              <span>URL file audio dùng chung cho cả bài (audio_url)</span>
              <input className="tb-input" value={listeningAudioUrl} onChange={(e) => setListeningAudioUrl(e.target.value)} />
            </label>
            <SectionsEditor sections={sections} onChange={setSections} />
          </>
        )}

        {error && <p className="tb-error">{error}</p>}
        {success && (
          <p className="tb-success">
            Đã tạo đề thành công (id: {success.id}).{' '}
            <button type="button" className="tb-link-btn" onClick={() => navigate(`/practice/${skill}`)}>
              Xem danh sách đề {skill}
            </button>
          </p>
        )}

        <Button type="submit" variant="primary" className="tb-submit-btn" disabled={submitting}>
          {submitting ? 'Đang tạo...' : 'Tạo đề'}
        </Button>
      </form>
    </div>
  );
}
