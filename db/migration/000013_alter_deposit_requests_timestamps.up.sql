-- Alter deposit_requests timestamp columns to timestamptz for proper timezone handling
ALTER TABLE deposit_requests 
  ALTER COLUMN expires_at TYPE timestamptz,
  ALTER COLUMN completed_at TYPE timestamptz,
  ALTER COLUMN created_at TYPE timestamptz,
  ALTER COLUMN updated_at TYPE timestamptz;
