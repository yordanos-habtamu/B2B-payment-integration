-- Migration: Create payment_events table
-- Description: Creates payment_events table for audit trail and event sourcing

-- Create payment_events table
CREATE TABLE IF NOT EXISTS payment_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    tenant_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    previous_status VARCHAR(20),
    new_status VARCHAR(20),
    event_data JSONB DEFAULT '{}',
    source VARCHAR(100) NOT NULL DEFAULT 'system',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT payment_events_event_type_check CHECK (event_type IN (
        'created', 'updated', 'processing_started', 'completed', 
        'failed', 'cancelled', 'retried', 'approved', 'rejected'
    ))
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_payment_events_payment_id ON payment_events(payment_id);
CREATE INDEX IF NOT EXISTS idx_payment_events_tenant_id ON payment_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_payment_events_event_type ON payment_events(event_type);
CREATE INDEX IF NOT EXISTS idx_payment_events_created_at ON payment_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_events_payment_created ON payment_events(payment_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_events_tenant_created ON payment_events(tenant_id, created_at DESC);

-- GIN index for event_data JSONB queries
CREATE INDEX IF NOT EXISTS idx_payment_events_data_gin ON payment_events USING GIN(event_data);

-- Add comments
COMMENT ON TABLE payment_events IS 'Payment event log for audit trail and event sourcing';
COMMENT ON COLUMN payment_events.id IS 'Unique identifier for event';
COMMENT ON COLUMN payment_events.payment_id IS 'Reference to the payment';
COMMENT ON COLUMN payment_events.tenant_id IS 'Tenant identifier for multi-tenancy';
COMMENT ON COLUMN payment_events.event_type IS 'Type of event that occurred';
COMMENT ON COLUMN payment_events.previous_status IS 'Previous payment status';
COMMENT ON COLUMN payment_events.new_status IS 'New payment status';
COMMENT ON COLUMN payment_events.event_data IS 'Additional event data in JSON format';
COMMENT ON COLUMN payment_events.source IS 'Source of the event (system, user, api, etc.)';
COMMENT ON COLUMN payment_events.created_at IS 'Timestamp when event occurred';
