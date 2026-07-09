import { useState } from 'react';
import QuestionGroupEditor from './QuestionGroupEditor';
import { QUESTION_TYPE_LABELS, emptyQuestionGroup } from '../../lib/questionTypes';

// GroupsListEditor manages the question_groups array within one
// passage/section: a picker for the type of the next group to add, plus
// the list of QuestionGroupEditor cards already added.
export default function GroupsListEditor({ groups, allowedTypes, onChange }) {
  const [nextType, setNextType] = useState(allowedTypes[0]);

  function addGroup() {
    onChange([...groups, emptyQuestionGroup(nextType)]);
  }

  function updateGroup(i, next) {
    onChange(groups.map((g, idx) => (idx === i ? next : g)));
  }

  function removeGroup(i) {
    onChange(groups.filter((_, idx) => idx !== i));
  }

  return (
    <div className="tb-groups-list">
      {groups.map((g, i) => (
        <QuestionGroupEditor
          key={i}
          group={g}
          allowedTypes={allowedTypes}
          onChange={(next) => updateGroup(i, next)}
          onRemove={() => removeGroup(i)}
        />
      ))}

      <div className="tb-add-group-row">
        <select className="tb-select" value={nextType} onChange={(e) => setNextType(e.target.value)}>
          {allowedTypes.map((t) => (
            <option key={t} value={t}>
              {QUESTION_TYPE_LABELS[t]}
            </option>
          ))}
        </select>
        <button type="button" className="tb-add-btn" onClick={addGroup}>
          + Thêm nhóm câu hỏi
        </button>
      </div>
    </div>
  );
}
