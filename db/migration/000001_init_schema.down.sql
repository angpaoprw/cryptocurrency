-- Drop tables in correct order to respect foreign key constraints
DROP TABLE IF EXISTS members;
DROP TABLE IF EXISTS member_groups;
DROP TABLE IF EXISTS admins;
DROP TABLE IF EXISTS admin_role_permissions;
DROP TABLE IF EXISTS admin_permissions;
DROP TABLE IF EXISTS admin_roles;
DROP TABLE IF EXISTS agents;