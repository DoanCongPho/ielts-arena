import './BandMeter.css';

const SEGMENTS = 18;

// Signature component (design-system.md §4). Same 18-segment shape used for
// a graded band score, XP progress, and question-group answer progress —
// only the labels around it change per context, never the fill math.
export default function BandMeter({ value = 0, max = 9, label, trailingLabel, className = '' }) {
  const ratio = max === 0 ? 0 : value / max;
  const filledCount = Math.max(0, Math.min(SEGMENTS, Math.round(ratio * SEGMENTS)));

  return (
    <div className={`ui-band-meter${className ? ` ${className}` : ''}`}>
      {(label || trailingLabel) && (
        <div className="ui-band-meter-labels">
          {label && <span className="ui-band-meter-label text-data-sm">{label}</span>}
          {trailingLabel && (
            <span className="ui-band-meter-trailing text-data-sm">{trailingLabel}</span>
          )}
        </div>
      )}
      <div className="ui-band-meter-track">
        {Array.from({ length: SEGMENTS }, (_, i) => (
          <div
            key={i}
            className={`ui-band-meter-segment${i < filledCount ? ' ui-band-meter-segment-filled' : ''}`}
          />
        ))}
      </div>
    </div>
  );
}
