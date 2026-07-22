import { frameImageURL } from '../../../lib/avatarFrame';
import './AvatarFrame.css';

// Wraps the existing 100-image PNG frame-art system (lib/avatarFrame.js) and
// layers the design system's ring treatment on top: gold if equipped, a
// plain outline if unlocked-but-not-equipped, no ring at all if locked
// (design-system.md §5 — locked frames should read as absent, not grayed out).
export default function AvatarFrame({
  imageUrl,
  frameLevel,
  state = 'unlocked',
  size = 96,
  className = '',
  ...props
}) {
  return (
    <span
      className={`ui-avatar-frame ui-avatar-frame-${state}${className ? ` ${className}` : ''}`}
      style={{ width: size, height: size }}
      {...props}
    >
      {imageUrl && <img className="ui-avatar-frame-photo" src={imageUrl} alt="" />}
      <img className="ui-avatar-frame-art" src={frameImageURL(frameLevel)} alt="" />
    </span>
  );
}
