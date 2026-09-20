DROP INDEX idx_submissions_grading_queue ON submissions;

ALTER TABLE submissions
    DROP COLUMN claimed_at,
    DROP COLUMN next_attempt_at,
    DROP COLUMN last_error,
    DROP COLUMN attempts;
