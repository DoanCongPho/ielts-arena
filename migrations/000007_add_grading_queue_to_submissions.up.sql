-- Turns the submissions table into the grading work queue. Grading used to
-- run inline in the HTTP request, so `status` only ever moved pending ->
-- graded/failed within a single call and the pending state was never
-- observable. These columns make the queue durable: a worker claims a row,
-- and a crash mid-grade leaves a lease that another worker reclaims.
ALTER TABLE submissions
    ADD COLUMN attempts        INT          NOT NULL DEFAULT 0 AFTER status,
    ADD COLUMN last_error      VARCHAR(500) NULL               AFTER attempts,
    ADD COLUMN next_attempt_at DATETIME(3)  NULL               AFTER last_error,
    ADD COLUMN claimed_at      DATETIME(3)  NULL               AFTER next_attempt_at;

-- The claim query filters on status and next_attempt_at on every poll.
CREATE INDEX idx_submissions_grading_queue ON submissions (status, next_attempt_at);
