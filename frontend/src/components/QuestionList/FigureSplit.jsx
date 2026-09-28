// FigureSplit puts a group's map or diagrams beside its questions, so the
// picture stays in view while answering instead of scrolling back up to
// it. The picture sticks as the questions scroll. Where there isn't room
// for two columns (a phone, or the reading page's question pane beside the
// passage) it falls back to the picture above the questions.
export default function FigureSplit({ images, alt, children }) {
  if (!images.length) return children;
  return (
    <div className="figure-split">
      <div className="figure-split-grid">
        <div className="figure-split-media">
          {images.map((src, i) => (
            <img key={src} className="figure-split-img" src={src} alt={images.length > 1 ? `${alt} ${i + 1}` : alt} />
          ))}
        </div>
        <div className="figure-split-body">{children}</div>
      </div>
    </div>
  );
}
