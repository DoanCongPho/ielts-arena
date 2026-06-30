CREATE TABLE IF NOT EXISTS writing_tests (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    skill VARCHAR(50) NOT NULL,
    task_type VARCHAR(50) NOT NULL,
    prompt TEXT NOT NULL,
    image_url VARCHAR(255) NULL,
    source VARCHAR(100) NULL,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    xp_gain INT NOT NULL DEFAULT 0,
    created_at DATETIME(3) NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS vocabulary (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    vocab VARCHAR(255) NOT NULL,
    meaning VARCHAR(255) NOT NULL,
    pronunciation VARCHAR(100) NULL,
    created_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_vocabulary_user_id (user_id),
    CONSTRAINT fk_vocabulary_user
        FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS writing_submissions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    writing_test_id BIGINT UNSIGNED NOT NULL,
    content TEXT NOT NULL,
    word_count INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    submitted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_writing_submissions_user_id (user_id),
    KEY idx_writing_submissions_writing_test_id (writing_test_id),
    CONSTRAINT fk_writing_submissions_user
        FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_writing_submissions_writing_test
        FOREIGN KEY (writing_test_id) REFERENCES writing_tests(id)
        ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS writing_scores (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    submission_id BIGINT UNSIGNED NOT NULL,
    task_response DECIMAL(2,1) NULL,
    coherence_cohesion DECIMAL(2,1) NULL,
    lexical_resource DECIMAL(2,1) NULL,
    grammar_range_accuracy DECIMAL(2,1) NULL,
    overall_score DECIMAL(2,1) NULL,
    feedback_task TEXT NULL,
    feedback_coherence TEXT NULL,
    feedback_lexical TEXT NULL,
    feedback_grammar TEXT NULL,
    model_answer TEXT NULL,
    graded_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_writing_scores_submission_id (submission_id),
    CONSTRAINT fk_writing_scores_submission
        FOREIGN KEY (submission_id) REFERENCES writing_submissions(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS corrections (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    writing_score_id BIGINT UNSIGNED NOT NULL,
    span TEXT NOT NULL,
    issue TEXT NOT NULL,
    suggestion TEXT NOT NULL,
    PRIMARY KEY (id),
    KEY idx_corrections_writing_score_id (writing_score_id),
    CONSTRAINT fk_corrections_writing_score
        FOREIGN KEY (writing_score_id) REFERENCES writing_scores(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
