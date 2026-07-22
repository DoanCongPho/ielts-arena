import GroupBlock from './GroupBlock';
import './QuestionList.css';

// QuestionList renders one passage's/section's question_groups. In live
// mode (no `results`) it's controlled via `answers`/`onChange`, keyed by
// each question's question_order. In review mode (`results` provided —
// used by SubmissionDetailPage) it's read-only, pre-filled from what was
// actually submitted, and decorated per-question with a correct/incorrect
// indicator.
export default function QuestionList({ groups, answers, onChange, disabled, results, highlights, onHighlightRemove, skill }) {
  return (
    <div className="question-list">
      {(groups || []).map((g) => (
        <GroupBlock
          key={g.group_order}
          group={g}
          answers={answers}
          onChange={onChange}
          disabled={disabled}
          results={results}
          highlights={highlights}
          onHighlightRemove={onHighlightRemove}
          skill={skill}
        />
      ))}
    </div>
  );
}
