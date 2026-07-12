ALTER TABLE users ADD COLUMN role VARCHAR(50) NOT NULL DEFAULT 'user';

ALTER TABLE refresh_tokens ADD COLUMN created_ip VARCHAR(100);
ALTER TABLE refresh_tokens ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE;

-- Backfill updated_at to created_at
UPDATE refresh_tokens SET updated_at = created_at WHERE updated_at IS NULL;

-- Make updated_at NOT NULL
ALTER TABLE refresh_tokens ALTER COLUMN updated_at SET NOT NULL;
