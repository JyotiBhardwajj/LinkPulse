-- Table to store user credentials and profile details.
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    -- Email is used for authentication and lookup; UNIQUE constraint automatically creates a B-Tree index.
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Index for soft-delete queries which filter records where deleted_at IS NULL.
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
