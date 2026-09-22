-- Groups tests by the book they come from ("cambridge" volume 20, test 1),
-- so the test list can show "Cambridge 20 -> Test 1..4". All three are
-- NULL for tests that don't belong to a series.
-- One ALTER per column: TiDB rejects an AFTER that names a column added in
-- the same statement.
ALTER TABLE tests ADD COLUMN series      VARCHAR(50) NULL AFTER task_type;
ALTER TABLE tests ADD COLUMN volume      INT         NULL AFTER series;
ALTER TABLE tests ADD COLUMN test_number INT         NULL AFTER volume;

CREATE INDEX idx_tests_series ON tests (skill, series, volume, test_number);
