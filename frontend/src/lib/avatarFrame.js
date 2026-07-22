// The 100 avatar-frame assets (frontend/public/avatar-frames/) are named
// 01.png .. 99.png, then literally 100.png — not a consistent zero-pad
// width, so the level-to-filename mapping has to special-case level 100.
export function frameImageURL(level) {
  const clamped = Math.min(100, Math.max(1, level));
  const file = clamped === 100 ? '100.png' : String(clamped).padStart(2, '0') + '.png';
  return `/avatar-frames/${file}`;
}
