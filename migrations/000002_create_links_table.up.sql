-- Table to store shortened link configurations.
CREATE TABLE IF NOT EXISTS links (
    id UUID PRIMARY KEY,
    original_url TEXT NOT NULL,
    -- Unique short code representing the redirection slug. Unique automatically indexes B-Tree.
    short_code VARCHAR(50) NOT NULL UNIQUE,
    title VARCHAR(255),
    -- Foreign key to users. Indexed explicitly below to speed up user dashboard queries.
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Index for routing redirects by short code.
CREATE INDEX IF NOT EXISTS idx_links_short_code ON links(short_code);

-- Index for retrieving links owned by a specific user.
CREATE INDEX IF NOT EXISTS idx_links_user_id ON links(user_id);

-- Index for soft-delete filtering queries.
CREATE INDEX IF NOT EXISTS idx_links_deleted_at ON links(deleted_at);
