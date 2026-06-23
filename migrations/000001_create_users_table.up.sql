CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    level INT NOT NULL DEFAULT 0,
    xp INT NOT NULL DEFAULT 0,
    rank_score INT NOT NULL DEFAULT 0,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;