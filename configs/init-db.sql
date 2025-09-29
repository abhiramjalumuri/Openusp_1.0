-- OpenUSP Database Initialization Script
-- This script is automatically executed when PostgreSQL container starts

-- Create extensions if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create database user with proper permissions (if not exists)
DO $$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'openusp') THEN
      CREATE ROLE openusp LOGIN PASSWORD 'openusp123';
   END IF;
END
$$;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE openusp_db TO openusp;
GRANT ALL ON SCHEMA public TO openusp;

-- Set timezone
SET timezone = 'UTC';

-- Create initial indexes for better performance (tables will be created by GORM migrations)
-- These will be created after the tables exist

\echo 'OpenUSP database initialization completed successfully!'