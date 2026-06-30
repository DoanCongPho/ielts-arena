DROP TABLE IF EXISTS corrections;
DROP TABLE IF EXISTS writing_scores;
DROP TABLE IF EXISTS writing_submissions;
DROP TABLE IF EXISTS writing_tests;

CREATE TABLE IF NOT EXISTS tests (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    skill VARCHAR(50) NOT NULL,
    task_type VARCHAR(50) NOT NULL,
    content_data JSON NOT NULL,
    source VARCHAR(100) NULL,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    xp_gain INT NOT NULL DEFAULT 0,
    created_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_tests_skill (skill)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS submissions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    test_id BIGINT UNSIGNED NOT NULL,
    payload JSON NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    submitted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_submissions_user_id (user_id),
    KEY idx_submissions_test_id (test_id),
    CONSTRAINT fk_submissions_user
        FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_submissions_test
        FOREIGN KEY (test_id) REFERENCES tests(id)
        ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS scores (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    submission_id BIGINT UNSIGNED NOT NULL,
    overall_band DECIMAL(2,1) NULL,
    details JSON NULL,
    graded_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_scores_submission_id (submission_id),
    CONSTRAINT fk_scores_submission
        FOREIGN KEY (submission_id) REFERENCES submissions(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
