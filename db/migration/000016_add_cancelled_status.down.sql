-- Revert: remove 'cancelled' from the withdrawal_requests status CHECK constraint
-- Note: This will fail if any rows have status='cancelled'. Update them first.
ALTER TABLE withdrawal_requests DROP CONSTRAINT IF EXISTS withdrawal_requests_status_check;
ALTER TABLE withdrawal_requests ADD CONSTRAINT withdrawal_requests_status_check 
    CHECK (status IN ('pending', 'processing', 'completed', 'failed'));
