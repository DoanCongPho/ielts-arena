import './SkillTag.css';

const SKILL_LABELS = {
  reading: 'Reading',
  writing: 'Writing',
  listening: 'Listening',
  speaking: 'Speaking',
};

export default function SkillTag({ skill, children, comingSoon = false, className = '' }) {
  return (
    <span
      className={
        `ui-skill-tag ui-skill-tag-${skill}` +
        (comingSoon ? ' ui-skill-tag-coming-soon' : '') +
        (className ? ` ${className}` : '')
      }
    >
      {children || SKILL_LABELS[skill] || skill}
    </span>
  );
}
