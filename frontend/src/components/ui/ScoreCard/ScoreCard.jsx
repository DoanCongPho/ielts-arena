import SkillTag from '../SkillTag/SkillTag';
import BandMeter from '../BandMeter/BandMeter';
import './ScoreCard.css';

// Shared band-score header for ScoreResult (Writing) and AutoGradeResult
// (Reading/Listening) — design-system.md §5 "Band score display". Each
// caller keeps its own distinct detail body (criteria/corrections vs.
// per-question list) and renders this for the header only.
export default function ScoreCard({ skill, band, secondaryLabel }) {
  return (
    <div className="ui-score-card">
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
