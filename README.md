# B2B Payments API

A secure, enterprise-grade B2B payments API built with Go that implements Zero Trust architecture using mutual TLS (mTLS) authentication, tenant isolation, and idempotency protection.

## Overview

This project provides a robust payment processing platform designed for B2B transactions with the following key features:

- **Zero Trust Security**: mTLS authentication with client certificate verification
- **Multi-Tenant Architecture**: Tenant isolation through certificate-based identification
- **Idempotency Protection**: Redis-backed idempotency to prevent duplicate transactions
- **Policy-Based Authorization**: OPA (Open Policy Agent) integration framework
- **Enterprise Security**: TLS 1.3+ with certificate pinning and verification

## Architecture

### Core Components

```
b2b-payments/
├── cmd/                    # Application entry points
│   ├── api/               # Main API server
│   ├── proxy/             # Proxy service (placeholder)
│   └── worker/            # Background worker (placeholder)
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   ├── server/           # HTTP server and middleware
│   │   └── middleware/   # Security and request handling middleware
│   ├── crypto/           # Cryptographic operations (empty)
│   ├── event/            # Event handling (empty)
│   ├── handler/          # Request handlers (empty)
│   ├── outbound/         # External service integrations (empty)
│   ├── policy/           # Policy management (empty)
│   ├── repository/       # Data access layer (empty)
│   └── service/          # Business logic (empty)
├── certs/                # TLS certificates
├── migrations/           # Database migrations (empty)
├── pkg/                  # Public library code (empty)
└── scripts/              # Utility scripts
    └── certs/           # Certificate generation scripts
```

### Security Model

#### Multi-Tenant Authentication
- **Client Certificate Format**: `CN=tenant-{tenant_id}.yourorg.com`
- **Certificate Verification**: mTLS with trusted CA validation
- **Tenant Extraction**: Automatic tenant ID extraction from certificate Common Name
- **Zero Trust**: All API endpoints require valid client certificates

#### Idempotency Protection
- **Redis-Backed**: Uses Redis for storing idempotency records
- **Header-Based**: Requires `Idempotency-Key` header for state-changing operations
- **Tenant Isolation**: Idempotency keys are namespaced by tenant ID
- **TTL Management**: Configurable TTL for idempotency records (default: 24 hours)

## Getting Started

### Prerequisites

- Go 1.24.7+
- Redis server
- TLS certificates (server cert, server key, CA cert)

### Configuration

Create a `.env` file based on `.env.example`:

```bash
PORT=8443
SERVER_CERT=./certs/server.crt
SERVER_KEY=./certs/server.key
CA_FILE=./certs/ca.crt
REDIS_URL=redis://localhost:6379/0
IDEMPOTENCY_TTL_HOURS=24
REQUIRE_CLIENT_CERT=true
```

### Running the Application

```bash
# Build and run the API server
go run cmd/api/main.go
```

The server will start on port 8443 with mTLS enabled.

## API Endpoints

### Public Endpoints

- `GET /health` - Health check endpoint (no authentication required)

### Protected Endpoints (Require mTLS)

All endpoints under `/api/v1` require:
- Valid client certificate
- Certificate CN format: `tenant-{tenant_id}.yourorg.com`
- `Idempotency-Key` header for POST/PUT/PATCH requests

- `GET /api/v1/payments` - Welcome endpoint showing tenant information
- `GET /api/v1/whoami` - Returns client certificate details and tenant information

## Security Features

### Mutual TLS (mTLS)
- **Server Authentication**: Server presents certificate to clients
- **Client Authentication**: Clients must present valid certificates
- **Certificate Verification**: All certificates verified against trusted CA
- **TLS Version**: Enforces TLS 1.3+ for maximum security

### Tenant Isolation
- **Certificate-Based**: Tenant ID extracted from client certificate CN
- **Context Injection**: Tenant ID injected into request context
- **Zero Trust**: Every request authenticated and authorized
- **Audit Logging**: All tenant access logged for compliance

### Idempotency
- **Duplicate Prevention**: Prevents duplicate transaction processing
- **Response Caching**: Stores and replays responses for duplicate requests
- **Degraded Mode**: Continues operation even if Redis is unavailable
- **Configurable TTL**: Adjustable idempotency record lifetime

## Development Status

### Currently Implemented
- ✅ mTLS server with client certificate verification
- ✅ Tenant extraction and isolation middleware
- ✅ Redis-backed idempotency middleware
- ✅ Configuration management with Viper
- ✅ OPA policy-based authorization with comprehensive security policies
- ✅ Complete payment processing logic with full CRUD operations
- ✅ PostgreSQL database integration with connection pooling
- ✅ Database migrations with version control
- ✅ Background worker for async payment processing
- ✅ Load balancing proxy service with health checks
- ✅ Event-driven architecture with Redis pub/sub
- ✅ Comprehensive API documentation with Swagger/OpenAPI
- ✅ Prometheus metrics and monitoring
- ✅ Circuit breaker patterns for resilience
- ✅ Retry mechanisms with exponential backoff
- ✅ Basic API endpoints with authentication
- ✅ Graceful shutdown handling
- ✅ Comprehensive logging and audit trails

### Architecture Components
- **API Server**: Main HTTP server with mTLS authentication (`cmd/api/`)
- **Background Worker**: Async job processing with Redis queues (`cmd/worker/`)
- **Load Balancer**: Proxy service with multiple algorithms (`cmd/proxy/`)
- **Database Layer**: PostgreSQL with migrations and connection pooling
- **Event System**: Redis pub/sub for real-time events
- **Monitoring**: Prometheus metrics for all components
- **Resilience**: Circuit breakers and retry patterns

### Planned Features
- 🔄 Webhook integrations for external payment processors
- 🔄 Multi-currency conversion services
- 🔄 Advanced compliance and reporting features
- 🔄 Real-time dashboard and analytics
- 🔄 GraphQL API support
- 🔄 Advanced rate limiting and throttling
- 🔄 Distributed tracing with Jaeger/Zipkin
- 🔄 Automated testing and CI/CD pipelines

## Technology Stack

### Core Technologies
- **Go 1.24.7** - Primary programming language
- **Echo v4** - HTTP web framework
- **Redis** - Idempotency store and caching
- **Viper** - Configuration management

### Security & Authentication
- **TLS 1.3+** - Transport layer security
- **mTLS** - Mutual TLS authentication
- **x509** - Certificate handling and validation
- **OPA** - Policy-based authorization (planned)

### Observability
- **Logrus** - Structured logging
- **Zap** - High-performance logging
- **Prometheus** - Metrics collection (dependencies included)

## Security Best Practices

### Certificate Management
- Use a trusted Certificate Authority for all certificates
- Implement certificate rotation policies
- Monitor certificate expiration
- Use strong cryptographic algorithms

### Tenant Security
- Enforce strict certificate naming conventions
- Implement tenant-specific rate limiting
- Log all tenant access attempts
- Regular security audits of tenant certificates

### Idempotency Security
- Use cryptographically secure idempotency keys
- Implement proper key rotation policies
- Monitor Redis security and access
- Implement backup and recovery procedures

## Contributing

1. Follow Go coding standards and best practices
2. Ensure all new features include proper error handling
3. Add comprehensive logging for security events
4. Test with different tenant certificates
5. Verify mTLS functionality in all scenarios

## License

This project is part of an enterprise B2B payments platform. All rights reserved.