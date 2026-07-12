ALTER TABLE users DROP COLUMN role;

ALTER TABLE refresh_tokens DROP COLUMN created_ip;
ALTER TABLE refresh_tokens DROP COLUMN updated_at;
