import { useState } from 'react';
import { buildVocabPrompt, VOCAB_COLUMNS } from '../../lib/vocab';
import './VocabPanel.css';

const CHATBOTS = [
  { name: 'ChatGPT', url: 'https://chatgpt.com' },
  { name: 'Claude', url: 'https://claude.ai' },
  { name: 'Gemini', url: 'https://gemini.google.com' },
];

// VocabPanel sits on a review: the words the learner highlighted (see
// useTextHighlights `words`), and a prompt to copy into a chatbot that
// turns them into a CSV for luyentu.com. The app calls no model itself.
// With no words yet, it tells the learner how to collect them.
export default function VocabPanel({ words, onRemove, source }) {
  const [copied, setCopied] = useState(false);
  const [open, setOpen] = useState(false);

  async function copyPrompt() {
    const prompt = buildVocabPrompt(words, source);
    try {
      await navigator.clipboard.writeText(prompt);
    } catch {
      // No clipboard access (an http page, an old browser): copy by hand.
      window.prompt('Sao chép prompt này (Ctrl/Cmd + C):', prompt);
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 2500);
  }

  if (words.length === 0) {
    return (
      <div className="vocab-panel vocab-panel-empty">
        <strong>📒 Lưu từ mới để học</strong>
        <p>
          Gặp từ chưa biết khi xem lại bài? Bôi đen từ đó rồi bấm <b>Tô đậm · lưu từ</b>. Các từ đã lưu hiện ở đây, kèm một
          prompt để tạo bộ thẻ từ vựng nhập vào luyentu.com.
        </p>
      </div>
    );
  }

  return (
    <div className="vocab-panel">
      <button type="button" className="vocab-panel-head" onClick={() => setOpen((o) => !o)} aria-expanded={open}>
        <span>📒 Từ vựng đã lưu ({words.length})</span>
        <span className="vocab-panel-caret">{open ? '▴' : '▾'}</span>
      </button>

      {open && (
        <div className="vocab-panel-body">
          <ul className="vocab-panel-words">
            {words.map((w) => (
              <li key={w.word.toLowerCase()} title={w.sentence}>
                {w.word}
                <button type="button" onClick={() => onRemove(w)} aria-label={`Bỏ ${w.word}`}>
                  ×
                </button>
              </li>
            ))}
          </ul>

          <button type="button" className="vocab-panel-copy" onClick={copyPrompt}>
            {copied ? '✓ Đã sao chép prompt' : 'Sao chép prompt tạo file từ vựng'}
          </button>

          <ol className="vocab-panel-steps">
            <li>
              Dán prompt vào{' '}
              {CHATBOTS.map((c, i) => (
                <span key={c.name}>
                  {i > 0 && (i === CHATBOTS.length - 1 ? ' hoặc ' : ', ')}
                  <a href={c.url} target="_blank" rel="noreferrer">
                    {c.name}
                  </a>
                </span>
              ))}
              .
            </li>
            <li>
              Tải file <code>tu-vung.csv</code> nó tạo ra (cột: {VOCAB_COLUMNS.join(', ')}). Kiểm tra nhanh nghĩa và phiên âm
              trước khi học.
            </li>
            <li>
              Trên{' '}
              <a href="https://luyentu.com" target="_blank" rel="noreferrer">
                luyentu.com
              </a>
              , chọn <b>Nhập file</b> và tải file này lên.
            </li>
          </ol>
        </div>
      )}
    </div>
  );
}
