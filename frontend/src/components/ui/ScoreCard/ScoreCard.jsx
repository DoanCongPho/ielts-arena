import SkillTag from '../SkillTag/SkillTag';
import BandMeter from '../BandMeter/BandMeter';
import './ScoreCard.css';

// Shared band-score header for ScoreResult (Writing) and AutoGradeResult
// (Reading/Listening) — design-system.md §5 "Band score display". Each
// caller keeps its own distinct detail body (criteria/corrections vs.
// per-question list) and renders this for the header only. `inline`
// lays it out as one short row, for review pages where the marked-up
// answers below are what the reader came for.
export default function ScoreCard({ skill, band, secondaryLabel, inline = false }) {
  return (
    <div className={`ui-score-card${inline ? ' ui-score-card-inline' : ''}`}>
      {skill && <SkillTag skill={skill} />}
      <span className="ui-score-card-value text-data-lg">{band}</span>
      <span className="ui-score-card-label text-label">Overall Band</span>
      <BandMeter value={Number(band) || 0} max={9} className="ui-score-card-meter" />
      {secondaryLabel && (
        <span className="ui-score-card-secondary text-body-sm">{secondaryLabel}</span>
      )}
    </div>
  );
}
