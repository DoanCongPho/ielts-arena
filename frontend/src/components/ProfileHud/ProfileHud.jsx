import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { getProfile } from '../../lib/api';
import { useChristmasTheme } from '../../lib/seasonalTheme';
import BandMeter from '../ui/BandMeter/BandMeter';
import Avatar from '../ui/Avatar/Avatar';
import { SantaHat } from '../ChristmasLayer/ornaments';
import './ProfileHud.css';

export default function ProfileHud() {
  const [profile, setProfile] = useState(null);
  const navigate = useNavigate();
  const christmas = useChristmasTheme();
  const hudRef = useRef(null);
  const loaded = profile != null;

  // Publish the HUD's size as CSS variables, so a page's pinned head can
  // sit beside it instead of under it (see .practice-top).
  useEffect(() => {
    const el = hudRef.current;
    if (!el) return undefined;
    const root = document.documentElement.style;
    const observer = new ResizeObserver(() => {
      const rect = el.getBoundingClientRect();
      root.setProperty('--profile-hud-height', `${Math.ceil(rect.height)}px`);
      root.setProperty('--profile-hud-right', `${Math.ceil(rect.right)}px`);
    });
    observer.observe(el);
    return () => {
      observer.disconnect();
      root.removeProperty('--profile-hud-height');
      root.removeProperty('--profile-hud-right');
    };
  }, [loaded]);

  useEffect(() => {
    let cancelled = false;

    function load() {
      getProfile()
        .then((data) => {
          if (!cancelled) setProfile(data);
        })
        .catch(() => {
          // Not authenticated, or the request failed — the HUD simply
          // doesn't render rather than surfacing an error to the user.
        });
    }

    load();
    window.addEventListener('profile:refresh', load);
    return () => {
      cancelled = true;
      window.removeEventListener('profile:refresh', load);
    };
  }, []);

  if (!profile) return null;

  const {
    name,
    level,
    current_level_xp: currentLevelXP,
    xp_to_next_level: xpToNextLevel,
    image_url: imageURL,
    xp_balance: xpBalance = 0,
    equipped_avatar: avatarId,
    equipped_frame: frameId,
  } = profile;

  const xpSpan = currentLevelXP + xpToNextLevel;

  return (
    <div className="profile-hud" ref={hudRef}>
      <div className="profile-hud-main">
        <span className="profile-hud-avatar-wrap">
          <Avatar name={name} imageURL={imageURL} avatarId={avatarId} frameId={frameId} size={frameId ? 68 : 56} />
          {/* A Santa hat in season, unless a frame is already dressing it up. */}
          {christmas.active && !frameId && <SantaHat size={34} className="profile-hud-hat" />}
        </span>
        <div className="profile-hud-info">
          <div className="profile-hud-name-row">
            <span className="profile-hud-level text-label">Lv. {level}</span>
            <span className="profile-hud-name text-body">{name}</span>
          </div>
          <BandMeter
            value={currentLevelXP}
            max={xpSpan || 1}
            trailingLabel={xpSpan === 0 ? 'Max level' : `${currentLevelXP} / ${xpSpan} XP`}
            className="profile-hud-xp-meter"
          />
          <div className="profile-hud-actions">
            <button
              type="button"
              className="profile-hud-wallet text-data-sm"
              onClick={() => navigate('/shop')}
              title="XP còn đổi được — mở cửa hàng"
            >
              <span aria-hidden="true">🎁</span> {xpBalance} XP · Đổi quà
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
