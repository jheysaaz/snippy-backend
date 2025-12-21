-- Rollback script for role system
-- Removes role tables and triggers

DROP TRIGGER IF EXISTS trigger_assign_default_role ON users;
DROP FUNCTION IF EXISTS assign_default_role();

DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
