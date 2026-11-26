import { useEffect } from 'react';
import { useChristmasTheme } from '../../lib/seasonalTheme';
import { Bauble, GiftBox, PineTree } from './ornaments';
import ThemeSwitch from './ThemeSwitch';
import './ChristmasLayer.css';

// Spread deterministically (golden-ratio steps) so the snow looks random
// but doesn't jump around between renders.
const FLAKES = Array.from({ length: 32 }, (_, i) => {
  const r = (i * 0.618034) % 1;
  return {
    left: `${(r * 100).toFixed(2)}%`,
    size: 4 + Math.round(((i * 7) % 9) * 0.75),
    duration: 11 + ((i * 5) % 9),
    delay: -((i * 3.7) % 18),
    drift: `${((i % 5) - 2) * 18}px`,
    opacity: 0.45 + ((i * 3) % 5) * 0.1,
  };
});

// ChristmasLayer switches the Christmas theme on for the pages it's
// mounted on (everything but the attempt pages, see App.jsx) by setting
// data-theme on <html>, and lets snow fall over them. It's taken off again
// on unmount, so an attempt page is always shown plain. Its on/off switch
// (ThemeSwitch) is shown on those same pages, theme on or off.
export default function ChristmasLayer() {
  const { active } = useChristmasTheme();

  useEffect(() => {
    if (!active) return undefined;
    const root = document.documentElement;
    root.dataset.theme = 'christmas';
    return () => {
      delete root.dataset.theme;
    };
  }, [active]);

  if (!active) return <ThemeSwitch />;

  return (
    <>
      <ThemeSwitch />
      <div className="xmas-snow" aria-hidden="true">
        {FLAKES.map((f, i) => (
          <span
            key={i}
            className="xmas-flake"
            style={{
              left: f.left,
              width: f.size,
              height: f.size,
              opacity: f.opacity,
              animationDuration: `${f.duration}s`,
              animationDelay: `${f.delay}s`,
              '--flake-drift': f.drift,
            }}
          />
        ))}
      </div>
    </>
  );
}

// ChristmasScenery is the snowy ground with trees and presents along the
// bottom of the roomy pages (dashboard, sign-in). Decoration only.
export function ChristmasScenery() {
  const { active } = useChristmasTheme();
  if (!active) return null;
  return (
    <div className="xmas-scenery" aria-hidden="true">
      <div className="xmas-scenery-side xmas-scenery-left">
        <PineTree size={150} />
        <PineTree size={96} />
        <GiftBox size={58} />
      </div>
      <div className="xmas-scenery-side xmas-scenery-right">
        <GiftBox size={64} />
        <Bauble size={44} />
        <PineTree size={118} />
        <PineTree size={170} />
      </div>
      <div className="xmas-scenery-ground" />
    </div>
  );
}
