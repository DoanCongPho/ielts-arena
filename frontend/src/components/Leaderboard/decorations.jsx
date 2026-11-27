// Drawings for the leaderboard's wooden board: the pine garland across
// its top, the snow on its upper edge, and the podium medals. Colors come
// from the theme tokens via inline styles.

const RED = 'var(--xmas-red-600)';
const RED_DARK = 'var(--xmas-red-800)';
const GREEN = 'var(--xmas-green-600)';
const GREEN_DARK = 'var(--xmas-green-800)';
const GOLD = 'var(--xmas-gold-500)';
const GOLD_DARK = 'var(--gold-600)';
const SNOW = 'var(--surface-0)';

// A point on the quadratic curve p0 -> p2 bent towards p1, at t in [0, 1].
function quad([x0, y0], [x1, y1], [x2, y2], t) {
  const u = 1 - t;
  return [u * u * x0 + 2 * u * t * x1 + t * t * x2, u * u * y0 + 2 * u * t * y1 + t * t * y2];
}

// Two swags of pine meeting under a bow in the middle, each with baubles
// hanging off it on strings.
const SWAGS = [
  [[0, 14], [250, 104], [500, 18]],
  [[500, 18], [750, 104], [1000, 14]],
];
const BAUBLE_COLORS = [RED, GOLD, 'var(--listening-500)', RED, GOLD, GREEN];

export function PineGarland({ className = '' }) {
  const path = SWAGS.map(([a, b, c], i) => `${i === 0 ? `M${a[0]} ${a[1]}` : ''} Q${b[0]} ${b[1]} ${c[0]} ${c[1]}`).join(' ');
  const baubles = SWAGS.flatMap((swag, s) =>
    [0.22, 0.5, 0.78].map((t, i) => {
      const [x, y] = quad(...swag, t);
      return { x, y, drop: i === 1 ? 34 : 22, color: BAUBLE_COLORS[s * 3 + i], r: i === 1 ? 13 : 10 };
    }),
  );
  const lights = SWAGS.flatMap((swag) => Array.from({ length: 9 }, (_, i) => quad(...swag, (i + 0.5) / 9)));

  return (
    <svg className={`lb-garland ${className}`.trim()} viewBox="0 0 1000 150" preserveAspectRatio="xMidYMin meet" aria-hidden="true">
      {/* Bushy pine: a dark base, then dashed lighter needles on top. */}
      <path d={path} style={{ stroke: GREEN_DARK, strokeWidth: 30, fill: 'none', strokeLinecap: 'round' }} />
      <path d={path} style={{ stroke: GREEN, strokeWidth: 24, fill: 'none', strokeDasharray: '3 4', strokeLinecap: 'round' }} />
      <path d={path} style={{ stroke: 'var(--xmas-green-fill)', strokeWidth: 12, fill: 'none', strokeDasharray: '1 9', strokeLinecap: 'round', opacity: 0.55 }} />
      {lights.map(([x, y], i) => (
        <circle
          key={`l${i}`}
          className="lb-garland-light"
          cx={x}
          cy={y + (i % 2 ? 6 : -6)}
          r="3.4"
          style={{ fill: [GOLD, RED, SNOW][i % 3], animationDelay: `${(i % 5) * 0.3}s` }}
        />
      ))}
      {baubles.map((b, i) => (
        <g key={`b${i}`} className="lb-garland-bauble" style={{ transformOrigin: `${b.x}px ${b.y}px`, animationDelay: `${i * 0.4}s` }}>
          <path d={`M${b.x} ${b.y} V${b.y + b.drop}`} style={{ stroke: GOLD_DARK, strokeWidth: 1.5 }} />
          <rect x={b.x - 4} y={b.y + b.drop - 3} width="8" height="6" rx="1.5" style={{ fill: GOLD_DARK }} />
          <circle cx={b.x} cy={b.y + b.drop + b.r + 1} r={b.r} style={{ fill: b.color }} />
          <ellipse cx={b.x - b.r * 0.35} cy={b.y + b.drop + b.r * 0.6} rx={b.r * 0.3} ry={b.r * 0.18} style={{ fill: SNOW, opacity: 0.6 }} />
        </g>
      ))}
      {/* The bow where the swags meet. */}
      <g transform="translate(500 22)">
        <path d="M0 0 C-30 -26 -62 -8 -44 10 C-34 20 -14 12 0 0 Z" style={{ fill: RED }} />
        <path d="M0 0 C30 -26 62 -8 44 10 C34 20 14 12 0 0 Z" style={{ fill: RED_DARK }} />
        <path d="M-4 4 L-22 48 L-12 42 L-6 54 Z" style={{ fill: RED }} />
        <path d="M4 4 L22 48 L12 42 L6 54 Z" style={{ fill: RED_DARK }} />
        <circle r="9" style={{ fill: RED }} />
        <circle r="4" cx="-2" cy="-2" style={{ fill: SNOW, opacity: 0.35 }} />
      </g>
    </svg>
  );
}

// SnowCap is the snow lying along the board's top edge, with drips.
export function SnowCap({ className = '' }) {
  return (
    <svg className={`lb-snowcap ${className}`.trim()} viewBox="0 0 1000 40" preserveAspectRatio="none" aria-hidden="true">
      <path
        d="M0 22 Q20 4 48 14 T110 12 T180 16 T250 8 T330 15 T410 9 T480 16 T560 8 T640 15 T720 9 T800 16 T880 8 T950 14 T1000 18
           V26 Q980 30 962 27 Q950 40 938 28 Q900 24 860 29 Q846 38 834 28 Q780 24 730 28 Q716 36 706 27 Q640 24 590 28
           Q574 39 562 27 Q500 23 440 28 Q426 35 416 27 Q350 24 300 28 Q286 40 274 27 Q210 23 160 28 Q146 36 136 27
           Q80 24 40 28 Q26 37 16 27 Q6 25 0 27 Z"
        style={{ fill: SNOW }}
      />
    </svg>
  );
}

const MEDAL = {
  1: { fill: GOLD, rim: GOLD_DARK, ribbon: [RED, RED_DARK] },
  2: { fill: 'var(--medal-silver)', rim: 'var(--ink-400)', ribbon: ['var(--listening-500)', 'var(--listening-600)'] },
  3: { fill: 'var(--medal-bronze)', rim: 'var(--wood-700)', ribbon: [GREEN, GREEN_DARK] },
};

// Medal for the podium's top three.
export function Medal({ rank, size = 44 }) {
  const m = MEDAL[rank];
  if (!m) return null;
  return (
    <svg className="lb-medal" width={size} height={size * 1.3} viewBox="0 0 40 52" aria-hidden="true">
      <path d="M8 0 H18 L24 20 H14 Z" style={{ fill: m.ribbon[0] }} />
      <path d="M32 0 H22 L16 20 H26 Z" style={{ fill: m.ribbon[1] }} />
      <circle cx="20" cy="34" r="16" style={{ fill: m.rim }} />
      <circle cx="20" cy="34" r="13" style={{ fill: m.fill }} />
      <text x="20" y="39.5" textAnchor="middle" style={{ fill: m.rim, fontSize: 15, fontWeight: 800, fontFamily: 'var(--font-display)' }}>
        {rank}
      </text>
      <ellipse cx="14" cy="28" rx="4" ry="2.4" transform="rotate(-30 14 28)" style={{ fill: SNOW, opacity: 0.5 }} />
    </svg>
  );
}

// Star for the top of the champion's portrait.
export function TopStar({ size = 34 }) {
  const pts = Array.from({ length: 10 }, (_, i) => {
    const r = i % 2 ? 7 : 16;
    const a = ((i * 36 - 90) * Math.PI) / 180;
    return `${(18 + r * Math.cos(a)).toFixed(2)},${(18 + r * Math.sin(a)).toFixed(2)}`;
  }).join(' ');
  return (
    <svg className="lb-top-star" width={size} height={size} viewBox="0 0 36 36" aria-hidden="true">
      <polygon points={pts} style={{ fill: GOLD, stroke: GOLD_DARK, strokeWidth: 1.5, strokeLinejoin: 'round' }} />
    </svg>
  );
}
