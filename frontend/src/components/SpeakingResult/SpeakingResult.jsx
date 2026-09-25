import { useEffect, useState } from 'react';
import { getRecordings } from '../../lib/api';
import { safeParse } from '../../lib/safeParse';
import ScoreCard from '../ui/ScoreCard/ScoreCard';
import SpeakingAnswerReview from './SpeakingAnswerReview';
import './SpeakingResult.css';

const CRITERIA = [
  'Fluency and Coherence',
  'Lexical Resource',
  'Grammatical Range and Accuracy',
  'Pronunciation',
];

const VERDICTS = { met: 'Đạt', partly: 'Một phần', not_met: 'Chưa đạt' };

// SpeakingResult shows a graded speaking test the way an examiner's report
// would: the four criterion bands with the descriptor evidence behind each,
// then the measurements and the transcript with the recordings.
export default function SpeakingResult({ score, submissionId }) {
  const d = safeParse(score.details) || {};
  const pron = d.pronunciation;
  const [recordings, setRecordings] = useState({});

  useEffect(() => {
    if (!submissionId) return;
    getRecordings(submissionId).then(setRecordings).catch(() => {});
  }, [submissionId]);

  return (
    <div className="speaking-result">
      <ScoreCard
        skill="speaking"
        band={score.overall_band}
        secondaryLabel={d.indicative ? 'Luyện một phần — band chỉ mang tính tham khảo' : undefined}
      />

      {(d.flags?.rehearsed?.length > 0 || d.flags?.off_topic?.length > 0) && (
        <p className="speaking-result-note">
          Một số câu trả lời nghe như học thuộc hoặc lạc đề nên không được tính khi chấm:{' '}
          {[...(d.flags.rehearsed || []), ...(d.flags.off_topic || [])].join(', ')}.
        </p>
      )}

      <div className="speaking-result-criteria">
        {CRITERIA.map((name) => {
          const c = d.criteria?.[name];
          if (!c) return null;
          return (
            <section key={name} className="speaking-result-criterion">
              <header className="speaking-result-criterion-header">
                <span>{name}</span>
                <span className="speaking-result-band">{c.score}</span>
              </header>
              {c.cap && (
                <p className="speaking-result-cap">
                  Band {c.judged_band} theo mô tả; giới hạn còn {c.score} — {c.cap.reason}.
                </p>
              )}
              {name === 'Pronunciation' && pron?.estimated && (
                <p className="speaking-result-cap">
                  Lần này không đo được phát âm trực tiếp, nên điểm được ước lượng từ độ rõ của lời nói.
                </p>
              )}
              <p className="speaking-result-feedback">{c.feedback}</p>
              {c.improvements?.length > 0 && (
                <ul className="speaking-result-improvements">
                  {c.improvements.map((tip) => <li key={tip}>{tip}</li>)}
                </ul>
              )}
              {c.checks?.length > 0 && (
                <details className="speaking-result-checks">
                  <summary>Đối chiếu với mô tả band</summary>
                  <table>
                    <tbody>
                      {c.checks.map((k, i) => (
                        <tr key={i}>
                          <td className="speaking-result-check-band">{k.band}</td>
                          <td>{k.feature}</td>
                          <td className={`speaking-result-verdict speaking-result-verdict-${k.verdict}`}>
                            {VERDICTS[k.verdict] || k.verdict}
                          </td>
                          <td className="speaking-result-check-evidence">{k.evidence}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </details>
              )}
            </section>
          );
        })}
      </div>

      <Measurements evidence={d.evidence} pron={pron} />

      {d.answers?.length > 0 && (
        <section className="speaking-result-section">
          <h3>Bài nói của bạn</h3>
          <div className="sa-legend">
            <span><span className="sa-word sa-word-bad">từ</span> phát âm sai</span>
            <span><span className="sa-word sa-word-fair">từ</span> chưa rõ</span>
            <span className="sa-fix sa-fix-grammar"><del>lỗi</del> <ins>sửa</ins></span>
            <span>Bấm vào một từ để nghe lại và so sánh từng âm.</span>
          </div>
          {d.answers.map((a) => (
            <div key={a.question_id} className="speaking-result-answer">
              <p className="speaking-result-question">
                <span className="text-label">Part {a.part}</span> {a.question_id === 'p2' ? a.question.split('\n')[0] : a.question}
              </p>
              <SpeakingAnswerReview
                answer={a}
                pronWords={pronWordsFor(pron, a.question_id)}
                corrections={(d.corrections || []).filter((c) => c.question_id === a.question_id)}
                recordingUrl={recordings[a.question_id]}
              />
            </div>
          ))}
        </section>
      )}
    </div>
  );
}

function Measurements({ evidence, pron }) {
  const f = evidence?.fluency;
  if (!f) return null;
  const stats = [
    ['Tốc độ nói', `${Math.round(f.speech_rate_wpm)} từ/phút`],
    ['Ngắt quãng ≥ 1 giây', `${f.long_pauses_per_min}/phút`],
    ['Số từ giữa hai lần ngừng', f.mean_length_of_run],
    ['Âm ngập ngừng (um, uh…)', `${f.fillers_per_100_words}/100 từ`],
    ['Lặp lại, sửa lời', `${f.repairs_per_100_words}/100 từ`],
  ];
  if (f.long_turn_seconds > 0) stats.push(['Độ dài Part 2', `${Math.round(f.long_turn_seconds)} giây`]);
  if (pron) {
    stats.push(['Từ phát âm rõ', `${Math.round(pron.intelligibility * 100)}%`]);
    if (!pron.estimated) {
      stats.push(['Biến đổi ngữ điệu', `${pron.prosody.pitch_std_st} semitone`]);
      stats.push(['Trọng âm từ đúng', `${Math.round(pron.prosody.stress_match_rate * 100)}%`]);
    }
  }
  return (
    <section className="speaking-result-section">
      <h3>Số liệu đo được</h3>
      <dl className="speaking-result-stats">
        {stats.map(([label, value]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
      {pron?.mispronounced?.length > 0 && (
        <>
          <h3>Từ cần luyện phát âm</h3>
          <ul className="speaking-result-words">
            {pron.mispronounced.map((w) => (
              <li key={w.word}>
                <strong>{w.word}</strong>
                {w.expected && (
                  <span className="speaking-result-ipa">
                    /{w.expected}/ {w.heard ? `→ nghe như /${w.heard}/` : '→ bị nuốt âm'}
                  </span>
                )}
                {w.count > 1 && <span className="speaking-result-count">×{w.count}</span>}
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  );
}

// pronWordsFor returns an answer's assessed words. Grades made before the
// word-by-word comparison only stored the flagged words' scores.
function pronWordsFor(pron, questionID) {
  if (!pron) return [];
  if (pron.words) return pron.words[questionID] || [];
  return (pron.by_question?.[questionID] || []).map((f) => ({ ...f, expected: '', heard: '', phonemes: [] }));
}
