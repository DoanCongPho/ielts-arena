DROP TABLE IF EXISTS user_items;
ALTER TABLE users
    DROP COLUMN equipped_frame,
    DROP COLUMN equipped_avatar,
    DROP COLUMN xp_spent;
