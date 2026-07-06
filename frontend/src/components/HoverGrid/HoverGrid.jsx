import { useEffect, useRef, useState, useCallback } from 'react';
import './HoverGrid.css';

const COLORS = ['#4c7cff', '#ffd447', '#34d399', '#9b8afb', '#ff7aa2'];
const CELL = 32;
const FADE_MS = 900;

export default function HoverGrid() {
  const containerRef = useRef(null);
  const [grid, setGrid] = useState({ cols: 0, rows: 0 });

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    let resizeTimeout;
    const compute = () => {
      const { width, height } = el.getBoundingClientRect();
      setGrid({
        cols: Math.ceil(width / CELL),
        rows: Math.ceil(height / CELL),
      });
    };
    const onResize = () => {
      clearTimeout(resizeTimeout);
      resizeTimeout = setTimeout(compute, 150);
    };

    const ro = new ResizeObserver(onResize);
    ro.observe(el);
    compute();

    return () => {
      ro.disconnect();
      clearTimeout(resizeTimeout);
    };
  }, []);

  const handleMouseOver = useCallback(e => {
    const cell = e.target.closest('.hover-grid-cell');
    if (!cell) return;
    const color = COLORS[Math.floor(Math.random() * COLORS.length)];
    cell.style.transition = 'none';
    cell.style.backgroundColor = color;
    // Force a reflow so the browser registers the instant color-in above
    // before the fade-out transition below is applied.
    void cell.offsetWidth;
    cell.style.transition = `background-color ${FADE_MS}ms ease-out`;
    cell.style.backgroundColor = 'transparent';
  }, []);

  const count = grid.cols * grid.rows;

  return (
    <div
      ref={containerRef}
      className="hover-grid"
      style={{ gridTemplateColumns: `repeat(${grid.cols}, ${CELL}px)` }}
      onMouseOver={handleMouseOver}
    >
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="hover-grid-cell" />
      ))}
    </div>
  );
}
