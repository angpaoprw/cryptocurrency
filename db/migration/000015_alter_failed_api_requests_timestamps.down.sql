-- Revert failed_api_requests timestamps back to timestamp
ALTER TABLE failed_api_requests 
  ALTER COLUMN created_at TYPE timestamp,
  ALTER COLUMN last_attempt TYPE timestamp;
