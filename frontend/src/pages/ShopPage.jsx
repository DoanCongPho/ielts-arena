import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { equipItem, getProfile, getShop, purchaseItem } from '../lib/api';
import Avatar from '../components/ui/Avatar/Avatar';
import Button from '../components/ui/Button/Button';
import ConfirmDialog from '../components/ui/ConfirmDialog/ConfirmDialog';
import { HollySprig } from '../components/ChristmasLayer/ornaments';
import './PracticePage.css';
import './ShopPage.css';

const TABS = [
  { kind: 'avatar', label: 'Avatar' },
  { kind: 'frame', label: 'Viền avatar' },
];

// ShopPage is where XP is spent: Christmas avatars and frames, previewed
// on the learner's own portrait. Spending never lowers the level — only
// the balance (lifetime XP minus purchases) goes down.
export default function ShopPage() {
  const navigate = useNavigate();
  const [shop, setShop] = useState(null);
  const [profile, setProfile] = useState(null);
  const [tab, setTab] = useState('avatar');
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [buying, setBuying] = useState(null); // the item awaiting confirmation
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    Promise.all([getShop(), getProfile()])
      .then(([s, p]) => {
        setShop(s);
        setProfile(p);
      })
      .catch((err) => setError(err.message));
  }, []);

  // Every change goes through the API, which answers with the whole shop;
  // the HUD reloads the profile to show the new balance and look.
  async function run(action, success) {
    setBusy(true);
    setError('');
    setNotice('');
    try {
      setShop(await action());
      window.dispatchEvent(new Event('profile:refresh'));
      if (success) setNotice(success);
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
      setBuying(null);
    }
  }

  if (!shop) {
    return (
      <div className="practice-page">
        <p className="practice-status">{error || 'Đang mở cửa hàng...'}</p>
      </div>
    );
  }

  const balance = shop.xp_balance;
  const items = shop.items.filter((it) => it.kind === tab);
  const portrait = { name: profile?.name, imageURL: profile?.image_url };

  function preview(it) {
    return it.kind === 'avatar'
      ? { avatarId: it.id, frameId: shop.equipped_frame }
      : { avatarId: shop.equipped_avatar, frameId: it.id };
  }

  function action(it) {
    if (it.equipped) {
      return (
        <Button variant="secondary" disabled={busy} onClick={() => run(() => equipItem(it.kind, ''), `Đã gỡ ${it.name}.`)}>
          Đang dùng · Gỡ ra
        </Button>
      );
    }
    if (it.owned) {
      return (
        <Button variant="primary" disabled={busy} onClick={() => run(() => equipItem(it.kind, it.id), `Đã dùng ${it.name}.`)}>
          Dùng
        </Button>
      );
    }
    if (!it.on_sale) {
      return (
        <Button variant="secondary" disabled title="Chỉ đổi được từ 1/12 đến 6/1">
          Hết mùa
        </Button>
      );
    }
    if (balance < it.price) {
      return (
        <Button variant="secondary" disabled>
          Thiếu {it.price - balance} XP
        </Button>
      );
    }
    return (
      <Button variant="primary" disabled={busy} onClick={() => setBuying(it)}>
        Đổi {it.price} XP
      </Button>
    );
  }

  return (
    <div className="practice-page shop-page">
      <div className="practice-top">
        <header className="practice-header">
          <h1 className="text-h1">Đổi XP</h1>
          <Button variant="secondary" onClick={() => navigate('/dashboard')}>
            ← Dashboard
          </Button>
        </header>
      </div>

      <section className="shop-hero">
        <Avatar
          {...portrait}
          avatarId={shop.equipped_avatar}
          frameId={shop.equipped_frame}
          size={112}
        />
        <div className="shop-hero-text">
          <span className="shop-hero-label text-label">XP còn đổi được</span>
          <span className="shop-hero-balance">{balance} XP</span>
          <p className="text-body-sm shop-hero-note">
            Đổi XP không làm giảm level — level tính theo tổng XP bạn đã kiếm. Mỗi đề làm lần đầu cho thêm XP.
          </p>
        </div>
        <HollySprig size={72} className="shop-hero-holly" />
      </section>

      <nav className="task-filters shop-tabs">
        {TABS.map((t) => (
          <button
            key={t.kind}
            type="button"
            className={`task-filter-pill ${tab === t.kind ? 'active' : ''}`}
            onClick={() => setTab(t.kind)}
          >
            {t.label}
          </button>
        ))}
      </nav>

      {error && <p className="practice-status practice-error">{error}</p>}
      {notice && <p className="practice-status shop-notice">{notice}</p>}

      <div className="shop-grid">
        {items.map((it) => (
          <article key={it.id} className={`shop-card ${it.equipped ? 'shop-card-equipped' : ''}`}>
            {it.season === 'christmas' && <span className="shop-card-season text-label">Giáng sinh</span>}
            <div className="shop-card-preview">
              <Avatar {...portrait} {...preview(it)} size={104} />
            </div>
            <h2 className="shop-card-name text-h3">{it.name}</h2>
            <span className="shop-card-price text-data-sm">
              {it.owned ? 'Đã sở hữu' : `${it.price} XP`}
            </span>
            {action(it)}
          </article>
        ))}
      </div>

      <ConfirmDialog
        open={!!buying}
        title="Đổi vật phẩm?"
        confirmLabel={buying ? `Đổi ${buying.price} XP` : 'Đổi'}
        cancelLabel="Để sau"
        busy={busy}
        onConfirm={() => run(() => purchaseItem(buying.id), `Đã đổi ${buying.name}! Bấm "Dùng" để mặc thử.`)}
        onCancel={() => setBuying(null)}
      >
        {buying && (
          <p>
            Dùng <strong>{buying.price} XP</strong> để đổi <strong>{buying.name}</strong>? Bạn sẽ còn{' '}
            {balance - buying.price} XP.
          </p>
        )}
      </ConfirmDialog>
    </div>
  );
}
