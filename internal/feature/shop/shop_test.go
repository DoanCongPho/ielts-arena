package shop

import (
	"context"
	"errors"
	"testing"
	"time"
)

var (
	december = func() time.Time { return time.Date(2026, time.December, 20, 10, 0, 0, 0, time.UTC) }
	november = func() time.Time { return time.Date(2026, time.November, 26, 10, 0, 0, 0, time.UTC) }
)

func newShop(xp int, clock func() time.Time, opts ...Option) (Service, *MockRepository) {
	repo := NewMockRepository()
	repo.Seed(1, xp)
	return NewService(repo, append([]Option{WithClock(clock)}, opts...)...), repo
}

func TestPurchaseSpendsBalanceButKeepsLifetimeXP(t *testing.T) {
	svc, repo := newShop(500, december)

	resp, err := svc.Purchase(context.Background(), 1, "frame-snowflake")
	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if resp.Balance != 300 {
		t.Errorf("balance = %d, want 300", resp.Balance)
	}
	// Lifetime XP — what the level comes from — is untouched.
	if repo.Wallets[1].XP != 500 {
		t.Errorf("lifetime XP = %d, want 500", repo.Wallets[1].XP)
	}
	for _, it := range resp.Items {
		if it.ID == "frame-snowflake" && !it.Owned {
			t.Error("the bought frame should be owned")
		}
	}
}

func TestPurchaseRejectsWhatCannotBeAfforded(t *testing.T) {
	svc, repo := newShop(100, december)

	_, err := svc.Purchase(context.Background(), 1, "frame-golden-star")
	if !errors.Is(err, ErrInsufficientXP) {
		t.Fatalf("err = %v, want ErrInsufficientXP", err)
	}
	if repo.Wallets[1].Spent != 0 || len(repo.Items[1]) != 0 {
		t.Error("a refused purchase must change nothing")
	}
}

func TestPurchaseTwiceIsRefused(t *testing.T) {
	svc, repo := newShop(1000, december)
	ctx := context.Background()

	if _, err := svc.Purchase(ctx, 1, "avatar-snowman"); err != nil {
		t.Fatalf("first purchase: %v", err)
	}
	if _, err := svc.Purchase(ctx, 1, "avatar-snowman"); !errors.Is(err, ErrAlreadyOwned) {
		t.Fatalf("second purchase err = %v, want ErrAlreadyOwned", err)
	}
	if repo.Wallets[1].Spent != 150 {
		t.Errorf("spent = %d, want 150 (charged once)", repo.Wallets[1].Spent)
	}
}

func TestChristmasItemsAreOnlyOnSaleInSeason(t *testing.T) {
	svc, _ := newShop(1000, november)
	if _, err := svc.Purchase(context.Background(), 1, "avatar-santa"); !errors.Is(err, ErrOutOfSeason) {
		t.Fatalf("err = %v, want ErrOutOfSeason", err)
	}

	forced, _ := newShop(1000, november, WithOpenSeasons(SeasonChristmas))
	if _, err := forced.Purchase(context.Background(), 1, "avatar-santa"); err != nil {
		t.Fatalf("with the season forced open: %v", err)
	}
}

func TestInSeasonWindow(t *testing.T) {
	cases := []struct {
		date time.Time
		want bool
	}{
		{time.Date(2026, time.November, 30, 23, 0, 0, 0, time.UTC), false},
		{time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2027, time.January, 6, 12, 0, 0, 0, time.UTC), true},
		{time.Date(2027, time.January, 7, 0, 0, 0, 0, time.UTC), false},
	}
	for _, c := range cases {
		if got := inSeason(SeasonChristmas, c.date); got != c.want {
			t.Errorf("inSeason(%s) = %v, want %v", c.date.Format("2006-01-02"), got, c.want)
		}
	}
}

func TestEquipNeedsOwnershipAndTheRightSlot(t *testing.T) {
	svc, _ := newShop(1000, december)
	ctx := context.Background()

	if _, err := svc.Equip(ctx, 1, KindFrame, "frame-lights"); !errors.Is(err, ErrNotOwned) {
		t.Fatalf("equip unowned: err = %v, want ErrNotOwned", err)
	}
	if _, err := svc.Purchase(ctx, 1, "frame-lights"); err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if _, err := svc.Equip(ctx, 1, KindAvatar, "frame-lights"); !errors.Is(err, ErrWrongSlot) {
		t.Fatalf("frame in avatar slot: err = %v, want ErrWrongSlot", err)
	}

	resp, err := svc.Equip(ctx, 1, KindFrame, "frame-lights")
	if err != nil {
		t.Fatalf("Equip: %v", err)
	}
	if resp.EquippedFrame == nil || *resp.EquippedFrame != "frame-lights" {
		t.Errorf("equipped frame = %v, want frame-lights", resp.EquippedFrame)
	}

	resp, err = svc.Equip(ctx, 1, KindFrame, "")
	if err != nil {
		t.Fatalf("unequip: %v", err)
	}
	if resp.EquippedFrame != nil {
		t.Errorf("equipped frame = %v after taking it off, want nil", *resp.EquippedFrame)
	}
}

// An item bought in season can still be worn after the season ends.
func TestOwnedItemsCanBeEquippedOutOfSeason(t *testing.T) {
	repo := NewMockRepository()
	repo.Seed(1, 1000)
	ctx := context.Background()

	if _, err := NewService(repo, WithClock(december)).Purchase(ctx, 1, "avatar-elf"); err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if _, err := NewService(repo, WithClock(november)).Equip(ctx, 1, KindAvatar, "avatar-elf"); err != nil {
		t.Fatalf("Equip out of season: %v", err)
	}
}
