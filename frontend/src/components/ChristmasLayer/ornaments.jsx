// Christmas ornaments drawn as inline SVG, on a 64×64 grid. Colors come
// from the theme tokens (tokens.css, :root[data-theme='christmas']) via
// inline `fill`/`stroke` styles — a var() is only reliable in CSS, not in
// an SVG presentation attribute.

const RED = 'var(--xmas-red-600)';
const RED_DARK = 'var(--xmas-red-800)';
const GREEN = 'var(--xmas-green-600)';
const GREEN_DARK = 'var(--xmas-green-800)';
const GOLD = 'var(--xmas-gold-500)';
const GOLD_DARK = 'var(--gold-600)';
const SNOW = 'var(--surface-0)';
const BROWN = 'var(--xmas-brown-600)';

const fill = (color) => ({ style: { fill: color } });
const stroke = (color, width) => ({ style: { stroke: color, strokeWidth: width, fill: 'none' } });

function Svg({ size = 64, className = '', title, children, viewBox = '0 0 64 64' }) {
  return (
    <svg
      className={`xmas-ornament ${className}`.trim()}
      width={size}
      height={size}
      viewBox={viewBox}
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : true}
      aria-label={title}
    >
      {children}
    </svg>
  );
}

export function PineTree(props) {
  return (
    <Svg {...props}>
      <rect x="28" y="50" width="8" height="10" rx="1.5" {...fill(BROWN)} />
      <path d="M32 18 L50 42 H14 Z" {...fill(GREEN_DARK)} />
      <path d="M32 12 L46 32 H18 Z" {...fill(GREEN)} />
      <path d="M32 26 L54 54 H10 Z" {...fill(GREEN)} />
      <path d="M32 6 l2.4 4.9 5.4.8-3.9 3.8.9 5.4-4.8-2.6-4.8 2.6.9-5.4-3.9-3.8 5.4-.8z" {...fill(GOLD)} />
      <circle cx="24" cy="44" r="2.6" {...fill(RED)} />
      <circle cx="38" cy="38" r="2.4" {...fill(GOLD)} />
      <circle cx="41" cy="49" r="2.6" {...fill(RED)} />
      <circle cx="28" cy="30" r="2.2" {...fill(RED)} />
      <circle cx="33" cy="47" r="2" {...fill(SNOW)} />
    </Svg>
  );
}

export function GiftBox(props) {
  return (
    <Svg {...props}>
      <rect x="10" y="26" width="44" height="32" rx="3" {...fill(RED)} />
      <rect x="7" y="19" width="50" height="11" rx="3" {...fill(RED_DARK)} />
      <rect x="28" y="19" width="8" height="39" {...fill(GOLD)} />
      <path d="M32 19 C24 6 12 10 18 17 C21 20 28 20 32 19 Z" {...fill(GOLD)} />
      <path d="M32 19 C40 6 52 10 46 17 C43 20 36 20 32 19 Z" {...fill(GOLD_DARK)} />
      <circle cx="32" cy="19" r="3.5" {...fill(GOLD_DARK)} />
      <circle cx="18" cy="40" r="1.6" {...fill(SNOW)} />
      <circle cx="46" cy="48" r="1.6" {...fill(SNOW)} />
      <circle cx="20" cy="51" r="1.2" {...fill(SNOW)} />
    </Svg>
  );
}

export function Bauble(props) {
  return (
    <Svg {...props}>
      <path d="M32 2 V12" {...stroke(GOLD_DARK, 1.5)} />
      <rect x="26" y="11" width="12" height="7" rx="2" {...fill(GOLD)} />
      <circle cx="32" cy="38" r="21" {...fill(RED)} />
      <path d="M12 34 Q32 44 52 34 L52.6 40 Q32 50 11.4 40 Z" {...fill(GOLD)} />
      <circle cx="22" cy="38" r="1.6" {...fill(RED_DARK)} />
      <circle cx="32" cy="41" r="1.6" {...fill(RED_DARK)} />
      <circle cx="42" cy="38" r="1.6" {...fill(RED_DARK)} />
      <ellipse cx="23" cy="27" rx="5" ry="3" transform="rotate(-35 23 27)" {...fill(SNOW)} opacity="0.55" />
    </Svg>
  );
}

export function Bell(props) {
  return (
    <Svg {...props}>
      <path d="M32 12 C20 12 17 24 17 34 C17 42 12 46 10 50 H54 C52 46 47 42 47 34 C47 24 44 12 32 12 Z" {...fill(GOLD)} />
      <path d="M32 12 C26 12 23 20 22.5 30" {...stroke(SNOW, 2.5)} opacity="0.5" />
      <rect x="9" y="48" width="46" height="5" rx="2.5" {...fill(GOLD_DARK)} />
      <circle cx="32" cy="56" r="4.5" {...fill(GOLD_DARK)} />
      <path d="M32 12 C26 2 16 4 20 10 C22 13 28 13 32 12 Z" {...fill(RED)} />
      <path d="M32 12 C38 2 48 4 44 10 C42 13 36 13 32 12 Z" {...fill(RED_DARK)} />
      <circle cx="32" cy="11" r="3" {...fill(RED)} />
    </Svg>
  );
}

export function Snowflake(props) {
  return (
    <Svg {...props}>
      <g style={{ stroke: 'currentColor', strokeWidth: 3, strokeLinecap: 'round', fill: 'none' }}>
        {[0, 60, 120].map((deg) => (
          <g key={deg} transform={`rotate(${deg} 32 32)`}>
            <path d="M32 6 V58" />
            <path d="M32 14 L26 8 M32 14 L38 8 M32 50 L26 56 M32 50 L38 56" />
            <path d="M32 22 L27 17 M32 22 L37 17 M32 42 L27 47 M32 42 L37 47" />
          </g>
        ))}
      </g>
    </Svg>
  );
}

export function HollySprig(props) {
  return (
    <Svg {...props}>
      <path
        d="M30 34 C26 30 20 31 18 26 C15 28 11 27 9 24 C11 22 11 18 8 16 C13 14 17 16 19 13 C21 17 26 17 28 20 C27 24 29 29 30 34 Z"
        {...fill(GREEN)}
      />
      <path
        d="M34 34 C38 30 44 31 46 26 C49 28 53 27 55 24 C53 22 53 18 56 16 C51 14 47 16 45 13 C43 17 38 17 36 20 C37 24 35 29 34 34 Z"
        {...fill(GREEN_DARK)}
      />
      <circle cx="28" cy="38" r="5" {...fill(RED)} />
      <circle cx="37" cy="38" r="5" {...fill(RED_DARK)} />
      <circle cx="32.5" cy="31" r="5" {...fill(RED)} />
      <circle cx="31" cy="29.5" r="1.4" {...fill(SNOW)} opacity="0.7" />
    </Svg>
  );
}

export function CandyCane(props) {
  return (
    <Svg {...props}>
      <path d="M38 60 V22 A10 10 0 0 0 18 22" style={{ stroke: SNOW, strokeWidth: 9, fill: 'none', strokeLinecap: 'round' }} />
      <path
        d="M38 60 V22 A10 10 0 0 0 18 22"
        style={{ stroke: RED, strokeWidth: 9, fill: 'none', strokeLinecap: 'round', strokeDasharray: '5 5' }}
      />
    </Svg>
  );
}

export function SantaHat(props) {
  return (
    <Svg {...props}>
      <path d="M10 46 C12 26 26 10 44 12 C52 13 56 22 54 30 C50 24 44 22 40 26 C46 32 50 40 52 46 Z" {...fill(RED)} />
      <path d="M40 26 C44 22 50 24 54 30" {...stroke(RED_DARK, 2)} />
      <rect x="6" y="42" width="50" height="12" rx="6" {...fill(SNOW)} />
      <circle cx="55" cy="32" r="6" {...fill(SNOW)} />
    </Svg>
  );
}

// Garland is a strand of lights: a drooping wire with bulbs that twinkle
// in turn (ChristmasLayer.css). It stretches to its container's width.
export function Garland({ bulbs = 14, className = '' }) {
  const colors = [RED, GOLD, GREEN, 'var(--listening-500)'];
  return (
    <div className={`xmas-garland ${className}`.trim()} aria-hidden="true">
      <svg className="xmas-garland-wire" viewBox="0 0 100 10" preserveAspectRatio="none">
        <path
          d={Array.from({ length: bulbs }, (_, i) => {
            const x0 = (i / bulbs) * 100;
            const x1 = ((i + 1) / bulbs) * 100;
            return `${i === 0 ? `M${x0} 1` : ''} Q${(x0 + x1) / 2} 9 ${x1} 1`;
          }).join(' ')}
          style={{ stroke: GREEN_DARK, strokeWidth: 0.6, fill: 'none' }}
          vectorEffect="non-scaling-stroke"
        />
      </svg>
      {Array.from({ length: bulbs }, (_, i) => (
        <span
          key={i}
          className="xmas-bulb"
          style={{
            left: `${((i + 0.5) / bulbs) * 100}%`,
            '--bulb-color': colors[i % colors.length],
            '--bulb-delay': `${(i % 4) * 0.4}s`,
          }}
        />
      ))}
    </div>
  );
}
