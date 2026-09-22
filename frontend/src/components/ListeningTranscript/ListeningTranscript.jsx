import HighlightableText from '../HighlightableText/HighlightableText';
import './ListeningTranscript.css';

// ListeningTranscript shows a section's transcript in a graded review, with
// the answer key's evidence highlighted. Paragraph ids follow the
// passage-{section}-p-{i} scheme useEvidenceReview scrolls to.
export default function ListeningTranscript({ section, paragraphs, evidenceFor }) {
  if (!paragraphs?.length) return null;
  return (
    <div className="listening-transcript">
      <h3 className="listening-transcript-title">Transcript</h3>
      <div className="attempt-passage-text listening-transcript-text">
        {paragraphs.map((p, pi) => (
          <p key={`${section}-${pi}`} id={`passage-${section}-p-${pi}`}>
            <HighlightableText id={`t-${pi}`} text={p.text} evidence={evidenceFor(section, pi)} />
          </p>
        ))}
      </div>
    </div>
  );
}
