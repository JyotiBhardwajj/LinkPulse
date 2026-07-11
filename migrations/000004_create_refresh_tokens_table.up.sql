-- Table to store hashed refresh tokens for user sessions.
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY,
    -- FK reference to the user. ON DELETE CASCADE ensures tokens are cleaned up if a user is deleted.
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- SHA-256 hash of the cryptographically random refresh token. Unique index prevents collisions.
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    device_name VARCHAR(100),
    ip_hash VARCHAR(64) NOT NULL,
    user_agent TEXT,
    last_used_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE
);

-- Index on token_hash is used for fast O(1) validations during refresh operations.
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);

-- Index on user_id allows efficient lookups to list active sessions or perform user-wide logouts (revoke all).
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
