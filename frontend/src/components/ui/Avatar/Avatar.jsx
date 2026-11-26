import { AvatarArt, FrameArt } from '../../Shop/items';
import './Avatar.css';

// Avatar is a user's portrait: a shop avatar if one is worn, else their
// photo, else their initial — inside the worn frame, if any. `size` is
// the outer box; with a frame the portrait shrinks to fit inside it.
export default function Avatar({ name, imageURL, avatarId, frameId, size = 56, className = '' }) {
  const portrait = avatarId ? (
    <AvatarArt id={avatarId} />
  ) : imageURL ? (
    <img src={imageURL} alt="" />
  ) : (
    <span className="ui-avatar-initial">{name?.charAt(0).toUpperCase()}</span>
  );

  return (
    <span
      className={`ui-avatar ${frameId ? 'ui-avatar-framed' : ''} ${className}`.trim()}
      style={{ width: size, height: size, fontSize: Math.round(size * 0.36) }}
      aria-hidden="true"
    >
      <span className="ui-avatar-portrait">{portrait}</span>
      {frameId && (
        <span className="ui-avatar-frame">
          <FrameArt id={frameId} />
        </span>
      )}
    </span>
  );
}
