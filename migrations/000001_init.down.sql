-- Drop initial schema for auth-service

DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP INDEX IF EXISTS idx_profiles_email;
DROP INDEX IF EXISTS idx_users_login;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS users;


