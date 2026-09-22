import { useEffect, useState } from 'react';
import { getProfile } from '../../lib/api';
import BandMeter from '../ui/BandMeter/BandMeter';
import './ProfileHud.css';

export default function ProfileHud() {
  const [profile, setProfile] = useState(null);

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
  } = profile;

  const xpSpan = currentLevelXP + xpToNextLevel;

  return (
    <div className="profile-hud">
      <div className="profile-hud-main">
        <span className="profile-hud-avatar" aria-hidden="true">
          {imageURL ? <img src={imageURL} alt="" /> : name?.charAt(0).toUpperCase()}
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
        </div>
      </div>
    </div>
  );
}
