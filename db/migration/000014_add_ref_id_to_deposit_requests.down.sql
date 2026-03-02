-- Remove ref_id column from deposit_requests table
DROP INDEX IF EXISTS idx_deposit_requests_ref_id;
ALTER TABLE deposit_requests 
  DROP COLUMN ref_id;
