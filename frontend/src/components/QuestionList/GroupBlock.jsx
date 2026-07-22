import ChoiceControl from './ChoiceControl';
import MatchingDragDrop from './MatchingDragDrop';
import PlainInputGroup from './PlainInputGroup';
import StructuredBlankGroup from './StructuredBlankGroup';
import HighlightableText from '../HighlightableText/HighlightableText';
import Card from '../ui/Card/Card';
import SkillTag from '../ui/SkillTag/SkillTag';

const CHOICE_TYPES = new Set(['true-false-not-given', 'yes-no-not-given', 'multiple-choice', 'multiple-choice-multi']);
const MATCHING_TYPES = new Set([
  'matching-headings',
  'matching-information',
  'matching-features',
  'matching-sentence-endings',
  'matching',
  'map-plan-labelling',
]);
const PLAIN_INPUT_TYPES = new Set(['sentence-completion', 'short-answer', 'diagram-label-completion']);
const STRUCTURED_TYPES = new Set(['summary-completion', 'table-completion', 'note-completion', 'flow-chart-completion', 'form-completion']);

function questionRangeLabel(questions) {
  const orders = questions.map((q) => q.question_order);
  const min = Math.min(...orders);
  const max = Math.max(...orders);
  return min === max ? `Câu ${min}` : `Câu ${min}-${max}`;
}

// GroupBlock renders one question_group's shared header (range + shared
// instructions), then dispatches to the control appropriate for its
// question_type.
export default function GroupBlock({ group, answers, onChange, disabled, results, highlights, onHighlightRemove, skill }) {
  const controlProps = { group, answers, onChange, disabled, results, highlights, onHighlightRemove };
  const instructionsKey = `group-${group.group_order}-instructions`;

  return (
    <Card padding="compact" className="question-group-block">
      <header className="question-group-header">
        <SkillTag skill={skill} className="question-group-range">
          {questionRangeLabel(group.questions)}
        </SkillTag>
        <HighlightableText
          as="p"
          className="question-group-instructions"
          id={instructionsKey}
          text={group.instructions}
          ranges={highlights?.[instructionsKey]}
          onRemoveRange={onHighlightRemove}
        />
      </header>

      {CHOICE_TYPES.has(group.question_type) && <ChoiceControl {...controlProps} />}
      {MATCHING_TYPES.has(group.question_type) && <MatchingDragDrop {...controlProps} />}
      {PLAIN_INPUT_TYPES.has(group.question_type) && <PlainInputGroup {...controlProps} />}
      {STRUCTURED_TYPES.has(group.question_type) && <StructuredBlankGroup {...controlProps} />}
    </Card>
  );
}
