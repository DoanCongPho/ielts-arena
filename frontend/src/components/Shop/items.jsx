// Art for the XP shop's items, keyed by the catalog IDs the API serves
// (internal/feature/shop/catalog.go). Avatars are drawn on a 64×64 grid
// filling a circle; frames on a 100×100 grid around a portrait of radius
// 36 at the centre. Colors come from the theme tokens via inline styles.

const RED = 'var(--xmas-red-600)';
const RED_DARK = 'var(--xmas-red-800)';
const GREEN = 'var(--xmas-green-600)';
const GREEN_DARK = 'var(--xmas-green-800)';
const GOLD = 'var(--xmas-gold-500)';
const GOLD_DARK = 'var(--gold-600)';
const SNOW = 'var(--surface-0)';
const BROWN = 'var(--xmas-brown-600)';
const SKIN = 'var(--xmas-skin)';
const ICE = 'var(--xmas-ice)';
const NIGHT = 'var(--xmas-night)';
const INK = 'var(--ink-900)';
const ORANGE = 'var(--speaking-500)';

const f = (color) => ({ style: { fill: color } });
const s = (color, width, extra = {}) => ({ style: { stroke: color, strokeWidth: width, fill: 'none', ...extra } });

// --- Avatars -----------------------------------------------------------

const AVATARS = {
  'avatar-snowman': (
    <>
      <rect width="64" height="64" {...f(NIGHT)} />
      <circle cx="12" cy="14" r="1.2" {...f(SNOW)} />
      <circle cx="52" cy="22" r="1" {...f(SNOW)} />
      <circle cx="46" cy="8" r="1.4" {...f(SNOW)} />
      <circle cx="32" cy="64" r="26" {...f(SNOW)} />
      <circle cx="32" cy="36" r="17" {...f(SNOW)} />
      <rect x="19" y="14" width="26" height="4" rx="1" {...f(INK)} />
      <rect x="23" y="2" width="18" height="14" rx="1.5" {...f(INK)} />
      <rect x="23" y="11" width="18" height="3" {...f(RED)} />
      <circle cx="26" cy="33" r="2" {...f(INK)} />
      <circle cx="38" cy="33" r="2" {...f(INK)} />
      <path d="M32 37 L44 40 L32 40.5 Z" {...f(ORANGE)} />
      {[24, 28, 32, 36, 40].map((x, i) => (
        <circle key={x} cx={x} cy={45 - Math.abs(i - 2) * 1.2} r="1.2" {...f(INK)} />
      ))}
      <path d="M14 54 Q32 60 50 54 L50 60 Q32 66 14 60 Z" {...f(RED)} />
    </>
  ),
  'avatar-gingerbread': (
    <>
      <rect width="64" height="64" {...f('var(--xmas-red-fill)')} />
      <circle cx="32" cy="64" r="24" {...f(BROWN)} />
      <circle cx="32" cy="30" r="18" {...f(BROWN)} />
      <path d="M17 28 q3 -3 6 0 q3 3 6 0 q3 -3 6 0 q3 3 6 0 q3 -3 6 0" {...s(SNOW, 1.6, { strokeLinecap: 'round' })} opacity="0.85" transform="translate(0 -12)" />
      <circle cx="25" cy="28" r="2.4" {...f(SNOW)} />
      <circle cx="39" cy="28" r="2.4" {...f(SNOW)} />
      <circle cx="22" cy="35" r="2.4" {...f(RED)} opacity="0.7" />
      <circle cx="42" cy="35" r="2.4" {...f(RED)} opacity="0.7" />
      <path d="M25 37 Q32 43 39 37" {...s(SNOW, 2.2, { strokeLinecap: 'round' })} />
      <circle cx="32" cy="52" r="2.6" {...f(RED)} />
      <circle cx="32" cy="60" r="2.6" {...f(GREEN)} />
      <path d="M22 48 q5 -3 0 -6" {...s(SNOW, 1.5, { strokeLinecap: 'round' })} />
      <path d="M42 48 q-5 -3 0 -6" {...s(SNOW, 1.5, { strokeLinecap: 'round' })} />
    </>
  ),
  'avatar-penguin': (
    <>
      <rect width="64" height="64" {...f(ICE)} />
      <ellipse cx="32" cy="44" rx="24" ry="28" {...f(INK)} />
      <ellipse cx="32" cy="38" rx="16" ry="17" {...f(SNOW)} />
      <circle cx="26" cy="33" r="2.4" {...f(INK)} />
      <circle cx="38" cy="33" r="2.4" {...f(INK)} />
      <circle cx="26.8" cy="32.2" r="0.8" {...f(SNOW)} />
      <circle cx="38.8" cy="32.2" r="0.8" {...f(SNOW)} />
      <path d="M28 38 L36 38 L32 43 Z" {...f(ORANGE)} />
      <path d="M10 52 Q32 60 54 52 L54 59 Q32 67 10 59 Z" {...f(RED)} />
      <rect x="40" y="55" width="7" height="12" rx="2" {...f(RED_DARK)} />
      <path d="M10 52 Q32 60 54 52" {...s(SNOW, 1.2, { strokeDasharray: '2 3' })} />
      <path d="M18 12 Q32 2 46 12 L44 16 Q32 8 20 16 Z" {...f(GREEN)} />
      <circle cx="32" cy="5" r="3.5" {...f(SNOW)} />
    </>
  ),
  'avatar-elf': (
    <>
      <rect width="64" height="64" {...f('var(--xmas-green-fill)')} />
      <path d="M8 34 L18 28 L18 40 Z" {...f(SKIN)} />
      <path d="M56 34 L46 28 L46 40 Z" {...f(SKIN)} />
      <circle cx="32" cy="38" r="15" {...f(SKIN)} />
      <path d="M14 30 Q20 4 48 2 Q40 14 50 30 Z" {...f(GREEN)} />
      <circle cx="49" cy="3" r="4" {...f(GOLD)} />
      <path d="M14 28 Q32 22 50 28 L50 33 Q32 27 14 33 Z" {...f(RED)} />
      <circle cx="26" cy="38" r="2" {...f(INK)} />
      <circle cx="38" cy="38" r="2" {...f(INK)} />
      <circle cx="22" cy="43" r="2.6" {...f(RED)} opacity="0.35" />
      <circle cx="42" cy="43" r="2.6" {...f(RED)} opacity="0.35" />
      <path d="M27 45 Q32 49 37 45" {...s(RED_DARK, 1.8, { strokeLinecap: 'round' })} />
      <path d="M12 64 Q16 52 32 53 Q48 52 52 64 Z" {...f(GREEN_DARK)} />
      <path d="M26 53 L32 60 L38 53" {...f(GOLD)} />
    </>
  ),
  'avatar-reindeer': (
    <>
      <rect width="64" height="64" {...f('var(--gold-100)')} />
      <path d="M20 22 L14 8 M14 8 L8 10 M14 8 L12 2 M17 15 L10 16" {...s(BROWN, 3, { strokeLinecap: 'round' })} />
      <path d="M44 22 L50 8 M50 8 L56 10 M50 8 L52 2 M47 15 L54 16" {...s(BROWN, 3, { strokeLinecap: 'round' })} />
      <ellipse cx="16" cy="28" rx="7" ry="4" transform="rotate(-25 16 28)" {...f(BROWN)} />
      <ellipse cx="48" cy="28" rx="7" ry="4" transform="rotate(25 48 28)" {...f(BROWN)} />
      <ellipse cx="32" cy="38" rx="15" ry="18" {...f(BROWN)} />
      <ellipse cx="32" cy="46" rx="10" ry="8" {...f('var(--gold-100)')} />
      <circle cx="26" cy="33" r="2.4" {...f(INK)} />
      <circle cx="38" cy="33" r="2.4" {...f(INK)} />
      <circle cx="32" cy="42" r="5" {...f(RED)} />
      <circle cx="30.5" cy="40.5" r="1.4" {...f(SNOW)} opacity="0.8" />
      <path d="M28 49 Q32 52 36 49" {...s(INK, 1.6, { strokeLinecap: 'round' })} />
      <path d="M18 58 Q32 64 46 58" {...s(GOLD_DARK, 2)} />
      <circle cx="32" cy="61" r="3" {...f(GOLD)} />
    </>
  ),
  'avatar-santa': (
    <>
      <rect width="64" height="64" {...f('var(--xmas-red-fill)')} />
      <path d="M12 26 Q14 6 36 6 Q50 7 54 22 Q48 16 42 18 Z" {...f(RED)} />
      <circle cx="55" cy="22" r="4.5" {...f(SNOW)} />
      <circle cx="32" cy="34" r="13" {...f(SKIN)} />
      <rect x="11" y="22" width="42" height="8" rx="4" {...f(SNOW)} />
      <path d="M16 36 Q18 62 32 62 Q46 62 48 36 Q44 46 32 46 Q20 46 16 36 Z" {...f(SNOW)} />
      <path d="M24 42 Q28 39 32 42 Q36 39 40 42 Q36 45 32 43 Q28 45 24 42 Z" {...f(SNOW)} />
      <circle cx="27" cy="34" r="1.8" {...f(INK)} />
      <circle cx="37" cy="34" r="1.8" {...f(INK)} />
      <circle cx="32" cy="38.5" r="2.6" {...f(RED)} opacity="0.55" />
      <circle cx="23" cy="38" r="2.4" {...f(RED)} opacity="0.3" />
      <circle cx="41" cy="38" r="2.4" {...f(RED)} opacity="0.3" />
    </>
  ),
};

// --- Frames ------------------------------------------------------------

const ring = (deg, r = 44) => {
  const a = ((deg - 90) * Math.PI) / 180;
  return [50 + r * Math.cos(a), 50 + r * Math.sin(a)];
};

function Flake({ x, y, size = 5 }) {
  return (
    <g transform={`translate(${x} ${y})`} {...s(SNOW, 1.4, { strokeLinecap: 'round' })}>
      {[0, 60, 120].map((d) => (
        <path key={d} d={`M0 ${-size} V${size}`} transform={`rotate(${d})`} />
      ))}
    </g>
  );
}

function Star({ x, y, r = 7, color = GOLD }) {
  const pts = Array.from({ length: 10 }, (_, i) => {
    const rad = i % 2 ? r * 0.45 : r;
    const a = (i * 36 - 90) * (Math.PI / 180);
    return `${(x + rad * Math.cos(a)).toFixed(2)},${(y + rad * Math.sin(a)).toFixed(2)}`;
  }).join(' ');
  return <polygon points={pts} style={{ fill: color, stroke: GOLD_DARK, strokeWidth: 0.8 }} />;
}

const FRAMES = {
  'frame-snowflake': (
    <>
      <circle cx="50" cy="50" r="42" {...s(ICE, 7)} />
      <circle cx="50" cy="50" r="45.5" {...s('var(--listening-500)', 1)} />
      {[0, 60, 120, 180, 240, 300].map((d) => {
        const [x, y] = ring(d, 42);
        return (
          <g key={d}>
            <circle cx={x} cy={y} r="6.5" {...f('var(--listening-500)')} />
            <Flake x={x} y={y} size={4.5} />
          </g>
        );
      })}
    </>
  ),
  'frame-candy-cane': (
    <>
      <circle cx="50" cy="50" r="43" {...s(SNOW, 8)} />
      <circle cx="50" cy="50" r="43" {...s(RED, 8, { strokeDasharray: '7 7' })} />
      <circle cx="50" cy="50" r="47.5" {...s(RED_DARK, 1)} />
      <circle cx="50" cy="50" r="38.5" {...s(RED_DARK, 1)} />
    </>
  ),
  'frame-ribbon': (
    <>
      <circle cx="50" cy="50" r="43" {...s(RED, 7)} />
      <circle cx="50" cy="50" r="43" {...s(GOLD, 1.5, { strokeDasharray: '1 4', strokeLinecap: 'round' })} />
      <path d="M50 8 C40 -6 22 0 30 8 C34 12 44 11 50 8 Z" {...f(GOLD)} />
      <path d="M50 8 C60 -6 78 0 70 8 C66 12 56 11 50 8 Z" {...f(GOLD_DARK)} />
      <path d="M47 9 L40 24 L46 21 L49 26 Z" {...f(GOLD)} />
      <path d="M53 9 L60 24 L54 21 L51 26 Z" {...f(GOLD_DARK)} />
      <circle cx="50" cy="8" r="4.5" {...f(GOLD_DARK)} />
    </>
  ),
  'frame-wreath': (
    <>
      <circle cx="50" cy="50" r="43" {...s(GREEN_DARK, 6)} />
      {Array.from({ length: 28 }, (_, i) => {
        const d = (i * 360) / 28;
        const [x, y] = ring(d, 43 + (i % 2 ? 3 : -3));
        return (
          <ellipse
            key={i}
            cx={x}
            cy={y}
            rx="6"
            ry="3"
            transform={`rotate(${d + (i % 2 ? 35 : -35)} ${x} ${y})`}
            {...f(i % 3 ? GREEN : GREEN_DARK)}
          />
        );
      })}
      {[20, 75, 130, 230, 285, 340].map((d) => {
        const [x, y] = ring(d, 44);
        return <circle key={d} cx={x} cy={y} r="2.6" {...f(RED)} />;
      })}
      <path d="M50 92 C40 104 30 96 38 90 Z" {...f(RED)} />
      <path d="M50 92 C60 104 70 96 62 90 Z" {...f(RED_DARK)} />
      <circle cx="50" cy="91" r="4" {...f(RED)} />
    </>
  ),
  'frame-lights': (
    <>
      <circle cx="50" cy="50" r="42" {...s(GREEN_DARK, 2.5)} />
      {Array.from({ length: 12 }, (_, i) => {
        const d = (i * 360) / 12;
        const [x, y] = ring(d, 44);
        const colors = [RED, GOLD, GREEN, 'var(--listening-500)'];
        return (
          <g key={i} transform={`rotate(${d} ${x} ${y})`}>
            <rect x={x - 2} y={y - 6} width="4" height="3" rx="0.6" {...f(GREEN_DARK)} />
            <ellipse
              className="shop-frame-bulb"
              cx={x}
              cy={y}
              rx="3.6"
              ry="5"
              style={{ fill: colors[i % 4], animationDelay: `${(i % 4) * 0.35}s` }}
            />
          </g>
        );
      })}
    </>
  ),
  'frame-golden-star': (
    <>
      <defs>
        <linearGradient id="shop-gold-ring" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="var(--xmas-gold-500)" />
          <stop offset="0.5" stopColor="var(--gold-100)" />
          <stop offset="1" stopColor="var(--gold-600)" />
        </linearGradient>
      </defs>
      <circle cx="50" cy="50" r="43" style={{ stroke: 'url(#shop-gold-ring)', strokeWidth: 7, fill: 'none' }} />
      <circle cx="50" cy="50" r="47" {...s(GOLD_DARK, 1)} />
      <Star x={50} y={6} r={10} />
      {[60, 130, 230, 300].map((d) => {
        const [x, y] = ring(d, 43);
        return <Star key={d} x={x} y={y} r={4.5} color={SNOW} />;
      })}
    </>
  ),
};

// AvatarArt draws a shop avatar filling its box (clipped to a circle by
// the caller). Unknown IDs draw nothing.
export function AvatarArt({ id }) {
  const art = AVATARS[id];
  if (!art) return null;
  return (
    <svg viewBox="0 0 64 64" width="100%" height="100%" aria-hidden="true" style={{ display: 'block' }}>
      {art}
    </svg>
  );
}

// FrameArt draws a frame over a box whose centre circle (radius 36 of 100)
// holds the portrait.
export function FrameArt({ id }) {
  const art = FRAMES[id];
  if (!art) return null;
  return (
    <svg viewBox="0 0 100 100" width="100%" height="100%" aria-hidden="true" style={{ display: 'block', overflow: 'visible' }}>
      {art}
    </svg>
  );
}
