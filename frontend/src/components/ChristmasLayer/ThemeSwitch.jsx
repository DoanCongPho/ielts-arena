import { useChristmasTheme } from '../../lib/seasonalTheme';
import { PineTree, Snowflake } from './ornaments';
import './ThemeSwitch.css';

// ThemeSwitch is the Christmas theme's on/off switch, styled like a
// dark/light-mode toggle and pinned to the top-right corner so it's easy
// to find. It works in or out of season (lib/seasonalTheme.js).
export default function ThemeSwitch() {
  const { active, setEnabled } = useChristmasTheme();
  const label = active ? 'Tắt giao diện Giáng sinh' : 'Bật giao diện Giáng sinh';

  return (
    <button
      type="button"
      role="switch"
      aria-checked={active}
      aria-label={label}
      title={label}
      className={`theme-switch ${active ? 'theme-switch-on' : ''}`}
      onClick={() => setEnabled(!active)}
    >
      <span className="theme-switch-track" aria-hidden="true">
        <span className="theme-switch-flake theme-switch-flake-a">
          <Snowflake size={10} />
        </span>
        <span className="theme-switch-flake theme-switch-flake-b">
          <Snowflake size={8} />
        </span>
        <span className="theme-switch-knob">
          {active ? <PineTree size={24} /> : <span className="theme-switch-sun" />}
        </span>
      </span>
      <span className="theme-switch-text">{active ? 'Noel' : 'Thường'}</span>
    </button>
  );
}
