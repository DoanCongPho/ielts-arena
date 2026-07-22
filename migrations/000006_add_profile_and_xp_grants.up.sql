ALTER TABLE users ADD COLUMN equipped_frame_level INT NULL DEFAULT NULL AFTER role;

CREATE TABLE IF NOT EXISTS submission_xp_grants (
    user_id BIGINT UNSIGNED NOT NULL,
    test_id BIGINT UNSIGNED NOT NULL,
    submission_id BIGINT UNSIGNED NOT NULL,
    xp_awarded INT NOT NULL,
    granted_at DATETIME(3) NOT NULL,
    PRIMARY KEY (user_id, test_id),
    CONSTRAINT fk_xp_grants_submission
        FOREIGN KEY (submission_id) REFERENCES submissions(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
