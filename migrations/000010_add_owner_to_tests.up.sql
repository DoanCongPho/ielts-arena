-- A test with an owner is a user's own practice content (a custom speaking
-- card), visible only to that user. Official tests, written by admins or
-- imported from books, have no owner.
ALTER TABLE tests ADD COLUMN owner_id BIGINT UNSIGNED NULL AFTER id;

CREATE INDEX idx_tests_owner ON tests (owner_id, skill, created_at);
