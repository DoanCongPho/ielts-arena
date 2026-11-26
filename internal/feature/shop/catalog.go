package shop

import "time"

// Kind is the slot an item goes in.
type Kind string

const (
	KindAvatar Kind = "avatar"
	KindFrame  Kind = "frame"
)

// Season limits when an item is on sale. An item bought in season is
// kept, and can be equipped, for good.
type Season string

const (
	SeasonNone      Season = ""
	SeasonChristmas Season = "christmas"
)

// Item is one thing in the shop. The catalog lives here rather than in a
// table: prices are a game-design decision shipped with the code, and the
// server — never the client — says what an item costs. The frontend draws
// each item by its ID (components/Shop/items.jsx).
type Item struct {
	ID     string `json:"id"`
	Kind   Kind   `json:"kind"`
	Name   string `json:"name"`
	Price  int    `json:"price"`
	Season Season `json:"season,omitempty"`
}

var catalog = []Item{
	{ID: "avatar-snowman", Kind: KindAvatar, Name: "Người tuyết", Price: 150, Season: SeasonChristmas},
	{ID: "avatar-gingerbread", Kind: KindAvatar, Name: "Bánh gừng", Price: 180, Season: SeasonChristmas},
	{ID: "avatar-penguin", Kind: KindAvatar, Name: "Chim cánh cụt quàng khăn", Price: 220, Season: SeasonChristmas},
	{ID: "avatar-elf", Kind: KindAvatar, Name: "Chú lùn tinh nghịch", Price: 260, Season: SeasonChristmas},
	{ID: "avatar-reindeer", Kind: KindAvatar, Name: "Tuần lộc mũi đỏ", Price: 320, Season: SeasonChristmas},
	{ID: "avatar-santa", Kind: KindAvatar, Name: "Ông già Noel", Price: 400, Season: SeasonChristmas},

	{ID: "frame-snowflake", Kind: KindFrame, Name: "Viền bông tuyết", Price: 200, Season: SeasonChristmas},
	{ID: "frame-candy-cane", Kind: KindFrame, Name: "Viền kẹo gậy", Price: 250, Season: SeasonChristmas},
	{ID: "frame-ribbon", Kind: KindFrame, Name: "Viền ruy băng quà", Price: 300, Season: SeasonChristmas},
	{ID: "frame-wreath", Kind: KindFrame, Name: "Vòng nguyệt quế", Price: 380, Season: SeasonChristmas},
	{ID: "frame-lights", Kind: KindFrame, Name: "Viền dây đèn", Price: 450, Season: SeasonChristmas},
	{ID: "frame-golden-star", Kind: KindFrame, Name: "Viền sao vàng lấp lánh", Price: 600, Season: SeasonChristmas},
}

// Catalog lists every item, in display order.
func Catalog() []Item {
	out := make([]Item, len(catalog))
	copy(out, catalog)
	return out
}

func findItem(id string) (Item, bool) {
	for _, it := range catalog {
		if it.ID == id {
			return it, true
		}
	}
	return Item{}, false
}

// inSeason reports whether a season's items are on sale at `now`.
// Christmas runs from 1 December to 6 January (Epiphany), the same window
// as the frontend's seasonal theme (lib/seasonalTheme.js).
func inSeason(s Season, now time.Time) bool {
	switch s {
	case SeasonNone:
		return true
	case SeasonChristmas:
		return now.Month() == time.December || (now.Month() == time.January && now.Day() <= 6)
	default:
		return false
	}
}
