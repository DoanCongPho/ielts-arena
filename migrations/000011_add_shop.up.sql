-- The XP shop. Lifetime XP (users.xp) still decides the level; buying
-- spends from a separate tally, so a purchase never costs a level. The
-- spendable balance is xp - xp_spent.
ALTER TABLE users
    ADD COLUMN xp_spent INT NOT NULL DEFAULT 0 AFTER xp,
    ADD COLUMN equipped_avatar VARCHAR(64) NULL AFTER image_url,
    ADD COLUMN equipped_frame VARCHAR(64) NULL AFTER equipped_avatar;

-- One row per item a user owns. The primary key makes buying the same
-- item twice impossible, whatever the request timing.
CREATE TABLE IF NOT EXISTS user_items (
    user_id BIGINT UNSIGNED NOT NULL,
    item_id VARCHAR(64) NOT NULL,
    price INT NOT NULL,
    purchased_at DATETIME(3) NOT NULL,
    PRIMARY KEY (user_id, item_id),
    CONSTRAINT fk_user_items_user
        FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
