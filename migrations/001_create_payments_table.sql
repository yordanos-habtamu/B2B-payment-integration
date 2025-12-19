-- Migration: Create payments table
-- Description: Creates the main payments table with all necessary indexes and constraints

-- Enable UUID extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create payments table
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(255) NOT NULL,
    amount DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL CHECK (currency IN ('USD', 'EUR', 'GBP')),
    type VARCHAR(10) NOT NULL CHECK (type IN ('credit', 'debit')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    description TEXT NOT NULL,
    reference VARCHAR(100),
    source_account VARCHAR(50) NOT NULL,
    destination_account VARCHAR(50) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    failure_reason TEXT,
    
    -- Constraints
    CONSTRAINT payments_source_dest_different CHECK (source_account != destination_account),
    CONSTRAINT payments_reference_unique UNIQUE (reference, tenant_id) WHERE reference IS NOT NULL
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_payments_tenant_id ON payments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_type ON payments(type);
CREATE INDEX IF NOT EXISTS idx_payments_currency ON payments(currency);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_tenant_status ON payments(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_payments_tenant_created ON payments(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_amount ON payments(amount);
CREATE INDEX IF NOT EXISTS idx_payments_source_account ON payments(source_account);
CREATE INDEX IF NOT EXISTS idx_payments_destination_account ON payments(destination_account);

-- GIN index for metadata JSONB queries
CREATE INDEX IF NOT EXISTS idx_payments_metadata_gin ON payments USING GIN(metadata);

-- Create a composite index for common queries
CREATE INDEX IF NOT EXISTS idx_payments_composite ON payments(tenant_id, status, created_at DESC);

-- Add comments for documentation
COMMENT ON TABLE payments IS 'Main payments table storing all B2B payment transactions';
COMMENT ON COLUMN payments.id IS 'Unique identifier for the payment';
COMMENT ON COLUMN payments.tenant_id IS 'Tenant identifier for multi-tenancy';
COMMENT ON COLUMN payments.amount IS 'Payment amount in the specified currency';
COMMENT ON COLUMN payments.currency IS 'ISO 4217 currency code';
COMMENT ON COLUMN payments.type IS 'Payment type: credit or debit';
COMMENT ON COLUMN payments.status IS 'Current payment status';
COMMENT ON COLUMN payments.description IS 'Human-readable payment description';
COMMENT ON COLUMN payments.reference IS 'External reference number (optional)';
COMMENT ON COLUMN payments.source_account IS 'Source account identifier';
COMMENT ON COLUMN payments.destination_account IS 'Destination account identifier';
COMMENT ON COLUMN payments.metadata IS 'Additional payment data in JSON format';
COMMENT ON COLUMN payments.created_at IS 'Timestamp when payment was created';
COMMENT ON COLUMN payments.updated_at IS 'Timestamp when payment was last updated';
COMMENT ON COLUMN payments.processed_at IS 'Timestamp when payment processing started';
COMMENT ON COLUMN payments.completed_at IS 'Timestamp when payment was completed';
COMMENT ON COLUMN payments.failed_at IS 'Timestamp when payment failed';
COMMENT ON COLUMN payments.failure_reason IS 'Reason for payment failure';

-- Create a function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
