-- 0019_user_avatar.sql: Add avatar_url to users for profile editing

ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
