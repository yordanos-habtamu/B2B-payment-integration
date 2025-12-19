package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{.Description}}",
        "title": "{{.Title}}",
        "termsOfService": "http://swagger.io/terms/",
        "contact": {
            "name": "API Support",
            "url": "http://www.swagger.io/support",
            "email": "support@swagger.io"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {},
    "definitions": {
        "Payment": {
            "type": "object",
            "required": [
                "id",
                "tenant_id",
                "amount",
                "currency",
                "type",
                "status",
                "description",
                "source_account",
                "destination_account",
                "created_at",
                "updated_at"
            ],
            "properties": {
                "amount": {
                    "type": "number",
                    "format": "double",
                    "example": 100.50
                },
                "completed_at": {
                    "type": "string",
                    "format": "date-time"
                },
                "created_at": {
                    "type": "string",
                    "format": "date-time"
                },
                "currency": {
                    "type": "string",
                    "enum": [
                        "USD",
                        "EUR",
                        "GBP"
                    ],
                    "example": "USD"
                },
                "description": {
                    "type": "string",
                    "example": "Payment for services rendered"
                },
                "destination_account": {
                    "type": "string",
                    "example": "dest-12345"
                },
                "failed_at": {
                    "type": "string",
                    "format": "date-time"
                },
                "failure_reason": {
                    "type": "string"
                },
                "id": {
                    "type": "string",
                    "example": "550e8400-e29b-41d4-a716-446655440000"
                },
                "metadata": {
                    "type": "object",
                    "additionalProperties": {}
                },
                "processed_at": {
                    "type": "string",
                    "format": "date-time"
                },
                "reference": {
                    "type": "string",
                    "example": "REF-12345"
                },
                "source_account": {
                    "type": "string",
                    "example": "src-12345"
                },
                "status": {
                    "type": "string",
                    "enum": [
                        "pending",
                        "processing",
                        "completed",
                        "failed",
                        "cancelled"
                    ],
                    "example": "pending"
                },
                "tenant_id": {
                    "type": "string",
                    "example": "tenant-123"
                },
                "type": {
                    "type": "string",
                    "enum": [
                        "credit",
                        "debit"
                    ],
                    "example": "credit"
                },
                "updated_at": {
                    "type": "string",
                    "format": "date-time"
                }
            }
        },
        "CreatePaymentRequest": {
            "type": "object",
            "required": [
                "amount",
                "currency",
                "type",
                "description",
                "source_account",
                "destination_account"
            ],
            "properties": {
                "amount": {
                    "type": "number",
                    "format": "double",
                    "example": 100.50
                },
                "currency": {
                    "type": "string",
                    "enum": [
                        "USD",
                        "EUR",
                        "GBP"
                    ],
                    "example": "USD"
                },
                "description": {
                    "type": "string",
                    "example": "Payment for services rendered"
                },
                "destination_account": {
                    "type": "string",
                    "example": "dest-12345"
                },
                "metadata": {
                    "type": "object",
                    "additionalProperties": {}
                },
                "reference": {
                    "type": "string",
                    "example": "REF-12345"
                },
                "source_account": {
                    "type": "string",
                    "example": "src-12345"
                },
                "type": {
                    "type": "string",
                    "enum": [
                        "credit",
                        "debit"
                    ],
                    "example": "credit"
                }
            }
        },
        "UpdatePaymentRequest": {
            "type": "object",
            "properties": {
                "description": {
                    "type": "string",
                    "example": "Updated payment description"
                },
                "metadata": {
                    "type": "object",
                    "additionalProperties": {}
                }
            }
        },
        "PaymentStats": {
            "type": "object",
            "properties": {
                "completed_amount": {
                    "type": "number",
                    "format": "double",
                    "example": 1500.75
                },
                "completed_count": {
                    "type": "integer",
                    "example": 15
                },
                "failed_amount": {
                    "type": "number",
                    "format": "double",
                    "example": 50.25
                },
                "failed_count": {
                    "type": "integer",
                    "example": 2
                },
                "pending_count": {
                    "type": "integer",
                    "example": 3
                },
                "total_amount": {
                    "type": "number",
                    "format": "double",
                    "example": 1601.00
                },
                "total_count": {
                    "type": "integer",
                    "example": 20
                }
            }
        },
        "HealthResponse": {
            "type": "object",
            "properties": {
                "status": {
                    "type": "string",
                    "example": "ok"
                }
            }
        },
        "ErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "Invalid request"
                }
            }
        },
        "WhoamiResponse": {
            "type": "object",
            "properties": {
                "cert_cn": {
                    "type": "string",
                    "example": "tenant-123.yourorg.com"
                },
                "cert_serial": {
                    "type": "string",
                    "example": "1234567890ABCDEF"
                },
                "dns_names": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "issuer": {
                    "type": "string",
                    "example": "YourOrg CA"
                },
                "not_after": {
                    "type": "string",
                    "format": "date-time"
                },
                "not_before": {
                    "type": "string",
                    "format": "date-time"
                },
                "tenant_id": {
                    "type": "string",
                    "example": "tenant-123"
                },
                "verified_chains": {
                    "type": "boolean",
                    "example": true
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8443",
	BasePath:         "/api/v1",
	Schemes:          []string{"https"},
	Title:            "B2B Payments API",
	Description:      "A secure, enterprise-grade B2B payments API with Zero Trust architecture",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
