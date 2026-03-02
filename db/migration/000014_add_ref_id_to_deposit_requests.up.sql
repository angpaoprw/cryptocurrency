-- Add ref_id column to deposit_requests table
ALTER TABLE deposit_requests 
  ADD COLUMN ref_id text;

-- Index for ref_id
CREATE INDEX idx_deposit_requests_ref_id ON deposit_requests(ref_id);
