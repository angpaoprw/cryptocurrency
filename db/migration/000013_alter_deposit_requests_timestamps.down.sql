-- Revert deposit_requests timestamp columns back to timestamp without timezone
ALTER TABLE deposit_requests 
  ALTER COLUMN expires_at TYPE timestamp,
  ALTER COLUMN completed_at TYPE timestamp,
  ALTER COLUMN created_at TYPE timestamp,
  ALTER COLUMN updated_at TYPE timestamp;
