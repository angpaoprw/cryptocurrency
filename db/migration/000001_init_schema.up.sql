-- Create tables
CREATE TABLE "agents" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "master_id" uuid REFERENCES "agents"("id") ON DELETE SET NULL,
  "name" text NOT NULL,
  "percentage" decimal(5, 2) NOT NULL DEFAULT 0 CHECK (percentage >= 0 AND percentage <= 100),
  "is_reseller" boolean NOT NULL DEFAULT false,
  "is_active" boolean NOT NULL DEFAULT true,
  "max_sub_agents" int DEFAULT 0 CHECK (max_sub_agents >= 0),
  "max_members" int DEFAULT 0 CHECK (max_members >= 0),
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now()),
  UNIQUE("name", "master_id"),
  CHECK (
    (is_reseller = true AND max_sub_agents > 0) OR 
    (is_reseller = false AND max_sub_agents = 0)
  )
);

CREATE TABLE "admin_roles" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "name" text NOT NULL,
  "description" text,
  "agent_id" uuid REFERENCES "agents"("id") ON DELETE CASCADE,
  "is_system_default" boolean NOT NULL DEFAULT false,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now()),
  UNIQUE("name", "agent_id"),
  CHECK (
    (agent_id IS NULL AND is_system_default = true) OR 
    (agent_id IS NOT NULL AND is_system_default = false)
  )
);

CREATE TABLE "admin_permissions" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "name" text NOT NULL,
  "resource" text NOT NULL,
  "action" text NOT NULL,
  "description" text,
  "agent_id" uuid REFERENCES "agents"("id") ON DELETE CASCADE,
  "is_system_default" boolean NOT NULL DEFAULT false,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  UNIQUE("name", "agent_id"),
  CHECK (
    (agent_id IS NULL AND is_system_default = true) OR 
    (agent_id IS NOT NULL AND is_system_default = false)
  )
);

CREATE TABLE "admin_role_permissions" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "role_id" uuid NOT NULL REFERENCES "admin_roles"("id") ON DELETE CASCADE,
  "permission_id" uuid NOT NULL REFERENCES "admin_permissions"("id") ON DELETE CASCADE,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  UNIQUE("role_id", "permission_id")
);

CREATE TABLE "admins" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "username" text NOT NULL,
  "password" text NOT NULL,
  "agent_id" uuid REFERENCES "agents"("id") ON DELETE SET NULL,
  "role_id" uuid REFERENCES "admin_roles"("id") ON DELETE SET NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "last_login" timestamp,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now()),
  UNIQUE("username", "agent_id")
);

CREATE TABLE "member_groups" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "name" text NOT NULL,
  "description" text,
  "agent_id" uuid REFERENCES "agents"("id") ON DELETE CASCADE,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now()),
  UNIQUE("name", "agent_id")
);

CREATE TABLE "members" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "username" text NOT NULL,
  "password" text NOT NULL,
  "agent_id" uuid REFERENCES "agents"("id") ON DELETE SET NULL,
  "group_id" uuid REFERENCES "member_groups"("id") ON DELETE SET NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "balance" decimal(15, 2) NOT NULL DEFAULT 0,
  "credit_limit" decimal(15, 2) NOT NULL DEFAULT 0,
  "last_login" timestamp,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now()),
  UNIQUE("username", "agent_id")
);

-- Insert System Default Permissions
INSERT INTO "admin_permissions" ("name", "resource", "action", "description", "agent_id", "is_system_default") VALUES
-- Admin permissions
('admin.create', 'admin', 'create', 'Create new admin accounts', NULL, true),
('admin.read', 'admin', 'read', 'View admin accounts', NULL, true),
('admin.update', 'admin', 'update', 'Update admin accounts', NULL, true),
('admin.delete', 'admin', 'delete', 'Delete admin accounts', NULL, true),

-- Agent permissions
('agent.create', 'agent', 'create', 'Create new agents', NULL, true),
('agent.read', 'agent', 'read', 'View agents', NULL, true),
('agent.update', 'agent', 'update', 'Update agents', NULL, true),
('agent.delete', 'agent', 'delete', 'Delete agents', NULL, true),

-- Member permissions
('member.create', 'member', 'create', 'Create new members', NULL, true),
('member.read', 'member', 'read', 'View members', NULL, true),
('member.update', 'member', 'update', 'Update members', NULL, true),
('member.delete', 'member', 'delete', 'Delete members', NULL, true),

-- Wallet permissions
('wallet.read', 'wallet', 'read', 'View wallet information', NULL, true),
('wallet.update', 'wallet', 'update', 'Update wallet balances', NULL, true),
('wallet.transaction', 'wallet', 'transaction', 'Process wallet transactions', NULL, true),

-- Promotion permissions
('promotion.create', 'promotion', 'create', 'Create promotions', NULL, true),
('promotion.read', 'promotion', 'read', 'View promotions', NULL, true),
('promotion.update', 'promotion', 'update', 'Update promotions', NULL, true),
('promotion.delete', 'promotion', 'delete', 'Delete promotions', NULL, true),

-- Rebate permissions
('rebate.read', 'rebate', 'read', 'View rebate information', NULL, true),
('rebate.update', 'rebate', 'update', 'Update rebate settings', NULL, true),
('rebate.calculate', 'rebate', 'calculate', 'Calculate rebates', NULL, true),

-- Game permissions
('game.read', 'game', 'read', 'View game information', NULL, true),
('game.config', 'game', 'config', 'Configure game settings', NULL, true),

-- Payment permissions
('payment.read', 'payment', 'read', 'View payment information', NULL, true),
('payment.process', 'payment', 'process', 'Process payments', NULL, true),

-- Reports permissions
('report.view', 'report', 'view', 'View reports', NULL, true),
('report.export', 'report', 'export', 'Export reports', NULL, true),

-- System permissions
('system.admin', 'system', 'admin', 'Full system administration access', NULL, true),
('system.config', 'system', 'config', 'System configuration access', NULL, true);

-- Insert System Default Roles
INSERT INTO "admin_roles" ("name", "description", "agent_id", "is_system_default") VALUES
('Super Admin', 'Full system access with all permissions', NULL, true),
('System Admin', 'System administration without agent management', NULL, true),
('Agent Manager', 'Can manage agents and their members', NULL, true),
('Member Manager', 'Can manage members and wallets only', NULL, true),
('Customer Support', 'Customer support with limited access', NULL, true),
('Financial Manager', 'Financial operations and reporting', NULL, true),
('Read Only', 'Read-only access to most resources', NULL, true);

-- Create role-permission relationships
-- Super Admin gets system.admin (which implies all permissions)
INSERT INTO "admin_role_permissions" ("role_id", "permission_id")
SELECT 
    (SELECT id FROM admin_roles WHERE name = 'Super Admin' AND is_system_default = true),
    (SELECT id FROM admin_permissions WHERE name = 'system.admin' AND is_system_default = true);

-- System Admin permissions
INSERT INTO "admin_role_permissions" ("role_id", "permission_id")
SELECT 
    (SELECT id FROM admin_roles WHERE name = 'System Admin' AND is_system_default = true),
    id
FROM admin_permissions 
WHERE name IN ('system.config', 'report.view', 'report.export', 'game.read', 'game.config', 'payment.read') 
AND is_system_default = true;

-- Agent Manager permissions
INSERT INTO "admin_role_permissions" ("role_id", "permission_id")
SELECT 
    (SELECT id FROM admin_roles WHERE name = 'Agent Manager' AND is_system_default = true),
    id
FROM admin_permissions 
WHERE name IN (
    'agent.create', 'agent.read', 'agent.update',
    'member.create', 'member.read', 'member.update',
    'wallet.read', 'wallet.update', 'wallet.transaction',
    'promotion.read', 'rebate.read', 'rebate.calculate', 'report.view'
) AND is_system_default = true;

-- Member Manager permissions  
INSERT INTO "admin_role_permissions" ("role_id", "permission_id")
SELECT 
    (SELECT id FROM admin_roles WHERE name = 'Member Manager' AND is_system_default = true),
    id
FROM admin_permissions 
WHERE name IN (
    'member.create', 'member.read', 'member.update',
    'wallet.read', 'wallet.update', 'wallet.transaction',
    'promotion.read', 'report.view'
) AND is_system_default = true;

-- Customer Support permissions
INSERT INTO "admin_role_permissions" ("role_id", "permission_id")
SELECT 
    (SELECT id FROM admin_roles WHERE name = 'Customer Support' AND is_system_default = true),
    id
FROM admin_permissions 
WHERE name IN (
    'member.read', 'member.update',
    'wallet.read', 'promotion.read',
    'payment.read', 'report.view'
) AND is_system_default = true;

-- Financial Manager permissions
INSERT INTO "admin_role_permissions" ("role_id", "permission_id")
SELECT 
    (SELECT id FROM admin_roles WHERE name = 'Financial Manager' AND is_system_default = true),
    id
FROM admin_permissions 
WHERE name IN (
    'wallet.read', 'wallet.update', 'wallet.transaction',
    'payment.read', 'payment.process',
    'rebate.read', 'rebate.update', 'rebate.calculate',
    'report.view', 'report.export'
) AND is_system_default = true;

-- Read Only permissions
INSERT INTO "admin_role_permissions" ("role_id", "permission_id")
SELECT 
    (SELECT id FROM admin_roles WHERE name = 'Read Only' AND is_system_default = true),
    id
FROM admin_permissions 
WHERE name IN (
    'agent.read', 'member.read', 'wallet.read',
    'promotion.read', 'rebate.read', 'game.read',
    'payment.read', 'report.view'
) AND is_system_default = true;