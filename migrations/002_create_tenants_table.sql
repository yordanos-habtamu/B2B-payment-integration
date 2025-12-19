-- Migration: Create tenants table
-- Description: Creates tenants table for multi-tenant management

-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
    tier VARCHAR(50) NOT NULL DEFAULT 'basic' CHECK (tier IN ('basic', 'professional', 'enterprise')),
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT tenants_name_unique UNIQUE (name)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
CREATE INDEX IF NOT EXISTS idx_tenants_tier ON tenants(tier);
CREATE INDEX IF NOT EXISTS idx_tenants_created_at ON tenants(created_at);

-- GIN index for settings and metadata
CREATE INDEX IF NOT EXISTS idx_tenants_settings_gin ON tenants USING GIN(settings);
CREATE INDEX IF NOT EXISTS idx_tenants_metadata_gin ON tenants USING GIN(metadata);

-- Add comments
COMMENT ON TABLE tenants IS 'Tenant information for multi-tenant architecture';
COMMENT ON COLUMN tenants.id IS 'Unique tenant identifier (matches certificate CN)';
COMMENT ON COLUMN tenants.name IS 'Human-readable tenant name';
COMMENT ON COLUMN tenants.status IS 'Tenant status: active, inactive, or suspended';
COMMENT ON COLUMN tenants.tier IS 'Subscription tier: basic, professional, or enterprise';
COMMENT ON COLUMN tenants.settings IS 'Tenant-specific settings in JSON format';
COMMENT ON COLUMN tenants.metadata IS 'Additional tenant metadata in JSON format';
COMMENT ON COLUMN tenants.created_at IS 'Timestamp when tenant was created';
COMMENT ON COLUMN tenants.updated_at IS 'Timestamp when tenant was last updated';

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
