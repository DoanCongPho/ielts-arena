import AnswerExplanation from '../AnswerExplanation/AnswerExplanation';
import HighlightableText from '../HighlightableText/HighlightableText';
import { optionLabel } from './chipPicker';

// The option text says nothing the letter doesn't ("Paragraph B" for B),
// so the column heading alone is enough.
function onlyNamesItsLetter(opt) {
  const text = String(opt.text ?? '').trim();
  return opt.id === text || new RegExp(`^(paragraph|section|part)\\s+${opt.id}$`, 'i').test(text);
}

// MatchingGrid answers a matching group whose letters may be used more than
// once the way the paper lays it out: a row per question, a column per
// letter, one choice per row. A bank of chips struck through as they are
// used would wrongly suggest a used letter is spent. Options whose text
// says more than their letter (a list of people, say) are listed above.
export default function MatchingGrid({ group, options, answers, onChange, disabled, results, highlights, onHighlightRemove }) {
  const legend = options.filter((o) => !onlyNamesItsLetter(o));

  return (
    <div className="matching-grid">
      {legend.length > 0 && (
        <ul className="matching-grid-legend">
          {legend.map((o) => (
            <li key={o.id}>{optionLabel(o)}</li>
          ))}
        </ul>
      )}

      <div className="matching-grid-scroll">
        <table className="matching-grid-table">
          <thead>
            <tr>
              <th aria-label="Câu hỏi" />
              {options.map((o) => (
                <th key={o.id} scope="col" title={o.text}>
                  {o.id}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {group.questions.map((q) => {
              const order = q.question_order;
              const result = results?.[order];
              const value = result ? result.submitted_answer?.[0] ?? '' : answers?.[order] ?? '';
              const correct = new Set(result?.correct_answer || []);
              const rowClass = result ? (result.correct ? 'matching-grid-correct' : 'matching-grid-incorrect') : '';
              const textKey = `q-${order}-text`;
              return (
                <tr key={order} id={`question-${order}`} className={rowClass}>
                  <th scope="row" className="matching-grid-question">
                    <span className="question-item-number">Câu {order}</span>{' '}
                    <HighlightableText id={textKey} text={q.text} ranges={highlights?.[textKey]} onRemoveRange={onHighlightRemove} />
                    {result && !result.correct && (
                      <span className="question-item-correct-answer matching-grid-answer">
                        Đáp án đúng: {(result.correct_answer || []).join(', ')}
                      </span>
                    )}
                    {result && <AnswerExplanation order={order} />}
                  </th>
                  {options.map((o) => (
                    <td key={o.id} className={result && correct.has(o.id) ? 'matching-grid-key' : ''}>
                      <input
                        type="radio"
                        name={`q-${order}`}
                        aria-label={`Câu ${order}: ${optionLabel(o)}`}
                        checked={value === o.id}
                        disabled={disabled}
                        onChange={() => onChange?.(order, o.id)}
                      />
                    </td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
