import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { deleteCustomSpeaking, listCustomSpeaking } from '../../lib/api';
import { safeParse } from '../../lib/safeParse';
import { MODE_LABELS, speakingSummary } from '../../lib/speakingContent';
import Button from '../ui/Button/Button';
import Card from '../ui/Card/Card';
import SkillTag from '../ui/SkillTag/SkillTag';
import './CustomSpeakingList.css';

// CustomSpeakingList is the "Đề của tôi" section on the Speaking page: the
// user's own tests, with a way to write a new one.
export default function CustomSpeakingList() {
  const navigate = useNavigate();
  const [tests, setTests] = useState([]);
  const [error, setError] = useState('');

  useEffect(() => {
    listCustomSpeaking()
      .then((data) => setTests(data.data || []))
      .catch((err) => setError(err.message));
  }, []);

  async function handleDelete(e, t) {
    e.stopPropagation();
    if (!window.confirm('Xoá đề này?')) return;
    try {
      await deleteCustomSpeaking(t.id);
      setTests((ts) => ts.filter((x) => x.id !== t.id));
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <section className="test-group">
      <div className="custom-speaking-header">
        <h2 className="test-group-title text-h3">Đề của tôi</h2>
        <Button variant="primary" onClick={() => navigate('/practice/speaking/custom/new')}>
          + Tạo đề của tôi
        </Button>
      </div>
      {error && <p className="practice-status practice-error">{error}</p>}
      {tests.length === 0 && !error && (
        <p className="practice-status">
          Tự viết cue card hay câu hỏi bạn muốn luyện — cả bài thi hoặc từng part — và được chấm như bài thi thật.
        </p>
      )}
      <div className="test-grid">
        {tests.map((t) => {
          const content = safeParse(t.content_data);
          return (
            <Card key={t.id} padding="compact" className="test-card" onClick={() => navigate(`/practice/speaking/${t.id}`)}>
              <div className="test-card-body">
                <div className="test-card-tags">
                  <SkillTag skill="speaking">{MODE_LABELS[t.task_type] || t.task_type}</SkillTag>
                  <span className="test-tag-neutral text-label">Của tôi</span>
                </div>
                <p className="test-card-prompt text-body-sm">{speakingSummary(content)}</p>
                <div className="custom-speaking-actions">
                  <button
                    type="button"
                    className="custom-speaking-link"
                    onClick={(e) => {
                      e.stopPropagation();
                      navigate(`/practice/speaking/custom/${t.id}/edit`);
                    }}
                  >
                    Sửa
                  </button>
                  <button type="button" className="custom-speaking-link" onClick={(e) => handleDelete(e, t)}>
                    Xoá
                  </button>
                </div>
              </div>
            </Card>
          );
        })}
      </div>
    </section>
  );
}
