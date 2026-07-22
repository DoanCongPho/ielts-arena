import { useEffect, useState } from 'react';
import { getProfile, setEquippedFrame } from '../../lib/api';
import AvatarFrame from '../ui/AvatarFrame/AvatarFrame';
import BandMeter from '../ui/BandMeter/BandMeter';
import './ProfileHud.css';

export default function ProfileHud() {
  const [profile, setProfile] = useState(null);
  const [pickerOpen, setPickerOpen] = useState(false);

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
    equipped_frame_level: equippedFrameLevel,
    unlocked_max_frame_level: unlockedMaxFrameLevel,
  } = profile;

  const xpSpan = currentLevelXP + xpToNextLevel;

  async function handlePickFrame(frameLevel) {
    if (frameLevel === equippedFrameLevel) {
      setPickerOpen(false);
      return;
    }
    try {
      const updated = await setEquippedFrame(frameLevel);
      setProfile(updated);
    } catch {
      // Leave the current selection in place on failure.
    }
    setPickerOpen(false);
  }

  const unlockedFrames = Array.from({ length: unlockedMaxFrameLevel }, (_, i) => i + 1);

  return (
    <div className="profile-hud">
      <div className="profile-hud-main">
        <button
          type="button"
          className="profile-hud-avatar-btn"
          onClick={() => setPickerOpen((open) => !open)}
          aria-label="Change avatar frame"
        >
          <AvatarFrame imageUrl={imageURL} frameLevel={equippedFrameLevel} state="equipped" size={96} />
        </button>
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

      {pickerOpen && (
        <div className="profile-hud-picker">
          {unlockedFrames.map((frameLevel) => (
            <button
              key={frameLevel}
              type="button"
              className="profile-hud-picker-item"
              onClick={() => handlePickFrame(frameLevel)}
              aria-label={`Equip frame ${frameLevel}`}
            >
              <AvatarFrame
                frameLevel={frameLevel}
                state={frameLevel === equippedFrameLevel ? 'equipped' : 'unlocked'}
                size={32}
              />
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
