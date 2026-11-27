import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { getLeaderboard } from '../lib/api';
import Avatar from '../components/ui/Avatar/Avatar';
import Button from '../components/ui/Button/Button';
import { CandyCane, GiftBox, HollySprig } from '../components/ChristmasLayer/ornaments';
import { Medal, PineGarland, SnowCap, TopStar } from '../components/Leaderboard/decorations';
import './PracticePage.css';
import './LeaderboardPage.css';

const SORTS = [
  { key: 'xp', label: 'Theo XP' },
  { key: 'attempts', label: 'Theo số bài' },
];

const SKILLS = [
  { key: 'reading', short: 'R', label: 'Reading' },
  { key: 'listening', short: 'L', label: 'Listening' },
  { key: 'writing', short: 'W', label: 'Writing' },
  { key: 'speaking', short: 'S', label: 'Speaking' },
];

const fmt = (n) => n.toLocaleString('vi-VN');

// LeaderboardPage is the "Bảng vàng": a wooden board hung with Christmas
// pine, ranking learners by lifetime XP or by tests finished, with each
// learner's tests per skill. The top three stand on a podium.
export default function LeaderboardPage() {
  const navigate = useNavigate();
  const [sort, setSort] = useState('xp');
  const [board, setBoard] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setError('');
    getLeaderboard(sort)
      .then((b) => !cancelled && setBoard(b))
      .catch((err) => !cancelled && setError(err.message));
    return () => {
      cancelled = true;
    };
  }, [sort]);

  const entries = board?.sort === sort ? board.entries : [];
  const podium = entries.slice(0, 3);
  const rest = entries.slice(3);

  const score = (e) => (sort === 'xp' ? `${fmt(e.xp)} XP` : `${fmt(e.attempts)} bài`);
  const subScore = (e) => (sort === 'xp' ? `${fmt(e.attempts)} bài` : `${fmt(e.xp)} XP`);

  return (
    <div className="practice-page leaderboard-page">
      <div className="practice-top">
        <header className="practice-header">
          <h1 className="text-h1">Bảng xếp hạng</h1>
          <Button variant="secondary" onClick={() => navigate('/dashboard')}>
            ← Dashboard
          </Button>
        </header>
      </div>

      <section className="lb-board" aria-label="Bảng xếp hạng">
        <SnowCap />
        <PineGarland />
        <span className="lb-nail lb-nail-tl" />
        <span className="lb-nail lb-nail-tr" />
        <span className="lb-nail lb-nail-bl" />
        <span className="lb-nail lb-nail-br" />
        <HollySprig size={64} className="lb-holly lb-holly-left" />
        <HollySprig size={64} className="lb-holly lb-holly-right" />

        <div className="lb-sign">
          <span className="lb-sign-rope lb-sign-rope-left" />
          <span className="lb-sign-rope lb-sign-rope-right" />
          <h2 className="lb-sign-title">Bảng Vàng</h2>
          <p className="lb-sign-sub">IELTS Arena · Mùa Giáng sinh</p>
        </div>

        <div className="lb-sorts" role="tablist">
          {SORTS.map((s) => (
            <button
              key={s.key}
              type="button"
              role="tab"
              aria-selected={sort === s.key}
              className={`lb-sort ${sort === s.key ? 'lb-sort-active' : ''}`}
              onClick={() => setSort(s.key)}
            >
              {s.label}
            </button>
          ))}
        </div>

        {error && <p className="lb-status lb-status-error">{error}</p>}
        {!error && !board && <p className="lb-status">Đang treo bảng...</p>}
        {board && entries.length === 0 && !error && (
          <p className="lb-status">Chưa có ai trên bảng — làm một đề để ghi tên mình nhé!</p>
        )}

        {podium.length > 0 && (
          <ol className="lb-podium">
            {podium.map((e) => (
              <li key={e.user_id} className={`lb-podium-spot lb-podium-${e.rank} ${e.is_me ? 'lb-is-me' : ''}`}>
                <div className="lb-podium-portrait">
                  {e.rank === 1 && <TopStar />}
                  <Avatar
                    name={e.name}
                    imageURL={e.image_url}
                    avatarId={e.equipped_avatar}
                    frameId={e.equipped_frame}
                    size={e.rank === 1 ? 104 : 84}
                  />
                  <Medal rank={e.rank} size={e.rank === 1 ? 40 : 34} />
                </div>
                <span className="lb-podium-name" title={e.name}>
                  {e.name}
                  {e.is_me && <span className="lb-me-tag">Bạn</span>}
                </span>
                <span className="lb-podium-level">Lv. {e.level}</span>
                <span className="lb-podium-score">{score(e)}</span>
                <span className="lb-podium-sub">{subScore(e)}</span>
                <SkillCounts skills={e.skills} />
                <div className="lb-podium-block">
                  <span>{e.rank}</span>
                </div>
              </li>
            ))}
          </ol>
        )}

        {rest.length > 0 && (
          <ol className="lb-list">
            {rest.map((e) => (
              <Row key={e.user_id} entry={e} score={score(e)} subScore={subScore(e)} />
            ))}
          </ol>
        )}

        {board?.me && board.sort === sort && (
          <>
            <div className="lb-gap" aria-hidden="true">• • •</div>
            <ol className="lb-list">
              <Row entry={board.me} score={score(board.me)} subScore={subScore(board.me)} />
            </ol>
          </>
        )}

        {board && entries.length > 0 && (
          <p className="lb-footnote">
            {fmt(board.total)} người trên bảng · chỉ tính bài đã chấm xong · XP là tổng XP đã kiếm (đổi quà không làm giảm)
          </p>
        )}

        <div className="lb-corner lb-corner-left" aria-hidden="true">
          <CandyCane size={58} />
          <GiftBox size={54} />
        </div>
        <div className="lb-corner lb-corner-right" aria-hidden="true">
          <GiftBox size={48} />
          <CandyCane size={58} />
        </div>
      </section>
    </div>
  );
}

function Row({ entry: e, score, subScore }) {
  return (
    <li className={`lb-row ${e.is_me ? 'lb-is-me' : ''}`}>
      <span className="lb-row-rank">{e.rank}</span>
      <Avatar
        name={e.name}
        imageURL={e.image_url}
        avatarId={e.equipped_avatar}
        frameId={e.equipped_frame}
        size={e.equipped_frame ? 52 : 44}
      />
      <span className="lb-row-who">
        <span className="lb-row-name" title={e.name}>
          {e.name}
          {e.is_me && <span className="lb-me-tag">Bạn</span>}
        </span>
        <span className="lb-row-level">Lv. {e.level}</span>
      </span>
      <SkillCounts skills={e.skills} />
      <span className="lb-row-score">
        <strong>{score}</strong>
        <small>{subScore}</small>
      </span>
    </li>
  );
}

function SkillCounts({ skills }) {
  return (
    <span className="lb-skills">
      {SKILLS.map((s) => (
        <span key={s.key} className={`lb-skill lb-skill-${s.key}`} title={`${s.label}: ${skills[s.key]} bài`}>
          <b>{s.short}</b>
          {skills[s.key]}
        </span>
      ))}
    </span>
  );
}
