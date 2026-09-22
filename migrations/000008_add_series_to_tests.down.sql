DROP INDEX idx_tests_series ON tests;
ALTER TABLE tests
    DROP COLUMN test_number,
    DROP COLUMN volume,
    DROP COLUMN series;
