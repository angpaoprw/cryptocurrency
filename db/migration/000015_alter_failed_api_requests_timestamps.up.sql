-- Update failed_api_requests timestamps to timestamptz
ALTER TABLE failed_api_requests 
  ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC',
  ALTER COLUMN last_attempt TYPE timestamptz USING last_attempt AT TIME ZONE 'UTC';
