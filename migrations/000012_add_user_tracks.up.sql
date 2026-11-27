-- Music a learner uploads for the Christmas music player. Only its
-- uploader can list or play a track. The audio lives in the media bucket
-- under music/<user_id>/; this row is what points at it.
CREATE TABLE IF NOT EXISTS user_tracks (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(200) NOT NULL,
    object_key VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size_bytes INT NOT NULL,
    created_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_user_tracks_user (user_id, created_at),
    CONSTRAINT fk_user_tracks_user
        FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
