import './IconChip.css';

const SKILL_ICONS = new Set(['reading', 'listening', 'writing', 'speaking']);

const ICON_PATHS = {
  reading: (
    <>
      <path d="M12 6.5c-1.6-1.1-4-1.6-6-1.1v12.2c2 0 4.4 0.6 6 1.6c1.6-1 4-1.6 6-1.6V5.4c-2-0.5-4.4 0-6 1.1z" />
      <path d="M12 6.5v12.2" />
    </>
  ),
  listening: (
    <>
      <path d="M4 13.5v-1.5a8 8 0 0 1 16 0v1.5" />
      <rect x="3" y="13.5" width="4.2" height="6" rx="1.6" />
      <rect x="16.8" y="13.5" width="4.2" height="6" rx="1.6" />
    </>
  ),
  writing: (
    <>
      <path d="M4 20l0.9-4.1L15.3 5.9a2 2 0 0 1 2.8 0l0.5 0.5a2 2 0 0 1 0 2.8L8.6 19.1z" />
      <path d="M13.6 7.4l3 3" />
    </>
  ),
  speaking: (
    <>
      <rect x="9" y="3" width="6" height="11" rx="3" />
      <path d="M5 11a7 7 0 0 0 14 0" />
      <path d="M12 18v3" />
      <path d="M9 21h6" />
    </>
  ),
  history: (
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="M12 7.5V12l3.2 2" />
    </>
  ),
  logout: (
    <>
      <path d="M9 4H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h3" />
      <path d="M15 8l4 4-4 4" />
      <path d="M19 12H9" />
    </>
  ),
  add: (
    <>
      <path d="M12 5v14" />
      <path d="M5 12h14" />
    </>
  ),
};

// A colored, iconified identity chip. Skill icons (reading/listening/
// writing/speaking) default to a gradient tone built from that skill's
// tokens; anything else (history, logout, add) defaults to a neutral
// hairline chip — see design-system.md §4b. Exists so skill identity can
// be conveyed visually (icon + color) without repeating the skill's name
// as text next to a label that already says it.
export default function IconChip({ icon, tone, size = 44, comingSoon = false, className = '' }) {
  const resolvedTone = tone || (SKILL_ICONS.has(icon) ? icon : 'neutral');
  const iconSize = Math.round(size * 0.5);

  return (
    <span
      className={
        `ui-icon-chip ui-icon-chip-${resolvedTone}` +
        (comingSoon ? ' ui-icon-chip-coming-soon' : '') +
        (className ? ` ${className}` : '')
      }
      style={{ width: size, height: size }}
    >
      <svg
        width={iconSize}
        height={iconSize}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        {ICON_PATHS[icon]}
      </svg>
    </span>
  );
}
