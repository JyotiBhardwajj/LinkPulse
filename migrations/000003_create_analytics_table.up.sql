-- Table to store append-only click tracking events.
CREATE TABLE IF NOT EXISTS analytics (
    id UUID PRIMARY KEY,
    -- Foreign key to links. Cascade delete ensures analytics are cleaned up if a link is deleted.
    link_id UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    clicked_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ip_hash VARCHAR(64) NOT NULL,
    country VARCHAR(100),
    city VARCHAR(100),
    browser VARCHAR(100),
    os VARCHAR(100),
    device VARCHAR(100),
    referrer TEXT,
    user_agent TEXT
);

-- Index to quickly lookup analytics for a specific link.
CREATE INDEX IF NOT EXISTS idx_analytics_link_id ON analytics(link_id);

-- Index for retrieving click events within a specific time range (e.g., last 24h, last 30d).
CREATE INDEX IF NOT EXISTS idx_analytics_clicked_at ON analytics(clicked_at);

-- Composite Index for analytics dashboard queries aggregating stats by link and timeframe.
-- Highly efficient for queries like: SELECT COUNT(*), browser FROM analytics WHERE link_id = ? AND clicked_at > ? GROUP BY browser;
CREATE INDEX IF NOT EXISTS idx_analytics_link_clicked ON analytics(link_id, clicked_at);
