-- Streak Database Initialization Script

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username VARCHAR(50) UNIQUE NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(20) NOT NULL CHECK (role IN ('athlete', 'follower')),
  display_name VARCHAR(100),
  bio TEXT,
  avatar_url VARCHAR(500),
  banner_url VARCHAR(500),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  is_verified BOOLEAN DEFAULT FALSE,
  is_banned BOOLEAN DEFAULT FALSE
);

-- Athlete profiles (extended info for athletes)
CREATE TABLE athlete_profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  sport_categories VARCHAR(50)[] NOT NULL DEFAULT '{}',
  stream_key VARCHAR(100) UNIQUE NOT NULL,
  rtmp_url VARCHAR(500),
  follower_count INTEGER DEFAULT 0,
  total_view_count BIGINT DEFAULT 0,
  is_live BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Streams table (metadata for each stream)
CREATE TABLE streams (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  athlete_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title VARCHAR(200) NOT NULL,
  description TEXT,
  sport_category VARCHAR(50) NOT NULL,
  tags VARCHAR(50)[] DEFAULT '{}',
  thumbnail_url VARCHAR(500),
  is_live BOOLEAN DEFAULT FALSE,
  started_at TIMESTAMP,
  ended_at TIMESTAMP,
  current_viewer_count INTEGER DEFAULT 0,
  peak_viewer_count INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stream sessions (individual broadcast sessions)
CREATE TABLE stream_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
  started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ended_at TIMESTAMP,
  duration_seconds INTEGER,
  average_viewers INTEGER DEFAULT 0,
  peak_viewers INTEGER DEFAULT 0,
  total_messages INTEGER DEFAULT 0
);

-- Follows relationship
CREATE TABLE follows (
  follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  athlete_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  notifications_enabled BOOLEAN DEFAULT TRUE,
  PRIMARY KEY (follower_id, athlete_id)
);

-- Indexes for performance
CREATE INDEX idx_streams_athlete_id ON streams(athlete_id);
CREATE INDEX idx_streams_is_live ON streams(is_live) WHERE is_live = TRUE;
CREATE INDEX idx_streams_sport_category ON streams(sport_category);
CREATE INDEX idx_streams_created_at ON streams(created_at DESC);
CREATE INDEX idx_follows_athlete_id ON follows(athlete_id);
CREATE INDEX idx_follows_follower_id ON follows(follower_id);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_athlete_profiles_is_live ON athlete_profiles(is_live) WHERE is_live = TRUE;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Function to increment follower count
CREATE OR REPLACE FUNCTION increment_follower_count()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE athlete_profiles
  SET follower_count = follower_count + 1
  WHERE user_id = NEW.athlete_id;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER after_follow_insert
  AFTER INSERT ON follows
  FOR EACH ROW
  EXECUTE FUNCTION increment_follower_count();

-- Function to decrement follower count
CREATE OR REPLACE FUNCTION decrement_follower_count()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE athlete_profiles
  SET follower_count = follower_count - 1
  WHERE user_id = OLD.athlete_id;
  RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER after_follow_delete
  AFTER DELETE ON follows
  FOR EACH ROW
  EXECUTE FUNCTION decrement_follower_count();

-- Insert some sample sport categories
COMMENT ON COLUMN streams.sport_category IS 'Sport categories: basketball, football, soccer, baseball, tennis, golf, running, cycling, swimming, hockey, martial-arts, esports, fitness, other';
COMMENT ON COLUMN athlete_profiles.sport_categories IS 'Array of sport categories this athlete streams';

-- Create a view for live streams with athlete info
CREATE VIEW live_streams_view AS
SELECT
  s.id,
  s.title,
  s.description,
  s.sport_category,
  s.tags,
  s.thumbnail_url,
  s.current_viewer_count,
  s.started_at,
  u.id as athlete_id,
  u.username as athlete_username,
  u.display_name as athlete_display_name,
  u.avatar_url as athlete_avatar_url,
  ap.follower_count as athlete_follower_count
FROM streams s
JOIN users u ON s.athlete_id = u.id
JOIN athlete_profiles ap ON u.id = ap.user_id
WHERE s.is_live = TRUE
ORDER BY s.current_viewer_count DESC;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO streak_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO streak_user;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO streak_user;
