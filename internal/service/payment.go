package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string
type PaymentType string
type Currency string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

const (
	PaymentTypeCredit PaymentType = "credit"
	PaymentTypeDebit  PaymentType = "debit"
)

const (
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyGBP Currency = "GBP"
)

type Payment struct {
	ID              string        `json:"id"`
	TenantID        string        `json:"tenant_id"`
	Amount          float64       `json:"amount"`
	Currency        Currency      `json:"currency"`
	Type            PaymentType   `json:"type"`
	Status          PaymentStatus `json:"status"`
	Description     string        `json:"description"`
	Reference       string        `json:"reference"`
	SourceAccount   string        `json:"source_account"`
	DestinationAccount string     `json:"destination_account"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	ProcessedAt     *time.Time    `json:"processed_at,omitempty"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
	FailedAt        *time.Time    `json:"failed_at,omitempty"`
	FailureReason   string        `json:"failure_reason,omitempty"`
}

type CreatePaymentRequest struct {
	Amount            float64                `json:"amount" validate:"required,gt=0"`
	Currency          Currency               `json:"currency" validate:"required"`
	Type              PaymentType            `json:"type" validate:"required,oneof=credit debit"`
	Description       string                 `json:"description" validate:"required,max=500"`
	Reference         string                 `json:"reference" validate:"max=100"`
	SourceAccount     string                 `json:"source_account" validate:"required,max=50"`
	DestinationAccount string               `json:"destination_account" validate:"required,max=50"`
	Metadata          map[string]interface{} `json:"metadata"`
}

type UpdatePaymentRequest struct {
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
	Metadata    *map[string]interface{} `json:"metadata,omitempty"`
}

type PaymentFilter struct {
	Status    *PaymentStatus `json:"status,omitempty"`
	Type      *PaymentType   `json:"type,omitempty"`
	Currency  *Currency      `json:"currency,omitempty"`
	MinAmount *float64       `json:"min_amount,omitempty"`
	MaxAmount *float64       `json:"max_amount,omitempty"`
	FromDate  *time.Time     `json:"from_date,omitempty"`
	ToDate    *time.Time     `json:"to_date,omitempty"`
	Limit     int            `json:"limit,omitempty"`
	Offset    int            `json:"offset,omitempty"`
}

type PaymentService interface {
	CreatePayment(ctx context.Context, tenantID string, req *CreatePaymentRequest) (*Payment, error)
	GetPayment(ctx context.Context, tenantID, paymentID string) (*Payment, error)
	UpdatePayment(ctx context.Context, tenantID, paymentID string, req *UpdatePaymentRequest) (*Payment, error)
	ListPayments(ctx context.Context, tenantID string, filter *PaymentFilter) ([]*Payment, int64, error)
	ProcessPayment(ctx context.Context, tenantID, paymentID string) error
	CancelPayment(ctx context.Context, tenantID, paymentID string) error
	GetPaymentStats(ctx context.Context, tenantID string) (*PaymentStats, error)
}

type PaymentStats struct {
	TotalCount      int64   `json:"total_count"`
	TotalAmount     float64 `json:"total_amount"`
	PendingCount    int64   `json:"pending_count"`
	CompletedCount  int64   `json:"completed_count"`
	FailedCount     int64   `json:"failed_count"`
	CompletedAmount float64 `json:"completed_amount"`
	FailedAmount    float64 `json:"failed_amount"`
}

type paymentService struct {
	// This will be implemented with repository pattern
	// For now, we'll create a basic in-memory implementation
	payments map[string]*Payment
}

func NewPaymentService() PaymentService {
	return &paymentService{
		payments: make(map[string]*Payment),
	}
}

func (s *paymentService) CreatePayment(ctx context.Context, tenantID string, req *CreatePaymentRequest) (*Payment, error) {
	// Validate the request
	if err := s.validateCreatePaymentRequest(tenantID, req); err != nil {
		return nil, err
	}

	// Generate payment ID
	paymentID := uuid.New().String()
	
	// Create payment
	payment := &Payment{
		ID:                 paymentID,
		TenantID:           tenantID,
		Amount:             req.Amount,
		Currency:           req.Currency,
		Type:               req.Type,
		Status:             PaymentStatusPending,
		Description:        req.Description,
		Reference:          req.Reference,
		SourceAccount:      req.SourceAccount,
		DestinationAccount: req.DestinationAccount,
		Metadata:           req.Metadata,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	// Store payment (in-memory for now)
	s.payments[paymentID] = payment

	return payment, nil
}

func (s *paymentService) GetPayment(ctx context.Context, tenantID, paymentID string) (*Payment, error) {
	payment, exists := s.payments[paymentID]
	if !exists {
		return nil, fmt.Errorf("payment not found")
	}

	// Verify tenant ownership
	if payment.TenantID != tenantID {
		return nil, fmt.Errorf("access denied")
	}

	return payment, nil
}

func (s *paymentService) UpdatePayment(ctx context.Context, tenantID, paymentID string, req *UpdatePaymentRequest) (*Payment, error) {
	payment, exists := s.payments[paymentID]
	if !exists {
		return nil, fmt.Errorf("payment not found")
	}

	// Verify tenant ownership
	if payment.TenantID != tenantID {
		return nil, fmt.Errorf("access denied")
	}

	// Only allow updates on pending payments
	if payment.Status != PaymentStatusPending {
		return nil, fmt.Errorf("payment cannot be updated in current status: %s", payment.Status)
	}

	// Update fields
	if req.Description != nil {
		payment.Description = *req.Description
	}
	
	if req.Metadata != nil {
		payment.Metadata = *req.Metadata
	}

	payment.UpdatedAt = time.Now().UTC()

	return payment, nil
}

func (s *paymentService) ListPayments(ctx context.Context, tenantID string, filter *PaymentFilter) ([]*Payment, int64, error) {
	var payments []*Payment
	
	for _, payment := range s.payments {
		if payment.TenantID != tenantID {
			continue
		}

		// Apply filters
		if filter != nil {
			if filter.Status != nil && payment.Status != *filter.Status {
				continue
			}
			if filter.Type != nil && payment.Type != *filter.Type {
				continue
			}
			if filter.Currency != nil && payment.Currency != *filter.Currency {
				continue
			}
			if filter.MinAmount != nil && payment.Amount < *filter.MinAmount {
				continue
			}
			if filter.MaxAmount != nil && payment.Amount > *filter.MaxAmount {
				continue
			}
			if filter.FromDate != nil && payment.CreatedAt.Before(*filter.FromDate) {
				continue
			}
			if filter.ToDate != nil && payment.CreatedAt.After(*filter.ToDate) {
				continue
			}
		}

		payments = append(payments, payment)
	}

	// Apply pagination
	total := int64(len(payments))
	if filter != nil && filter.Limit > 0 {
		offset := filter.Offset
		if offset >= len(payments) {
			return []*Payment{}, total, nil
		}
		
		end := offset + filter.Limit
		if end > len(payments) {
			end = len(payments)
		}
		
		payments = payments[offset:end]
	}

	return payments, total, nil
}

func (s *paymentService) ProcessPayment(ctx context.Context, tenantID, paymentID string) error {
	payment, exists := s.payments[paymentID]
	if !exists {
		return fmt.Errorf("payment not found")
	}

	// Verify tenant ownership
	if payment.TenantID != tenantID {
		return fmt.Errorf("access denied")
	}

	// Only process pending payments
	if payment.Status != PaymentStatusPending {
		return fmt.Errorf("payment cannot be processed in current status: %s", payment.Status)
	}

	// Update status to processing
	payment.Status = PaymentStatusProcessing
	payment.UpdatedAt = time.Now().UTC()
	now := time.Now().UTC()
	payment.ProcessedAt = &now

	// Simulate payment processing (in real implementation, this would integrate with payment processors)
	// For demo purposes, we'll randomly succeed or fail
	// In a real system, this would be an async operation
	
	// For now, let's mark as completed
	payment.Status = PaymentStatusCompleted
	payment.CompletedAt = &now
	payment.UpdatedAt = time.Now().UTC()

	return nil
}

func (s *paymentService) CancelPayment(ctx context.Context, tenantID, paymentID string) error {
	payment, exists := s.payments[paymentID]
	if !exists {
		return fmt.Errorf("payment not found")
	}

	// Verify tenant ownership
	if payment.TenantID != tenantID {
		return fmt.Errorf("access denied")
	}

	// Only cancel pending or processing payments
	if payment.Status != PaymentStatusPending && payment.Status != PaymentStatusProcessing {
		return fmt.Errorf("payment cannot be cancelled in current status: %s", payment.Status)
	}

	payment.Status = PaymentStatusCancelled
	payment.UpdatedAt = time.Now().UTC()

	return nil
}

func (s *paymentService) GetPaymentStats(ctx context.Context, tenantID string) (*PaymentStats, error) {
	stats := &PaymentStats{}

	for _, payment := range s.payments {
		if payment.TenantID != tenantID {
			continue
		}

		stats.TotalCount++
		stats.TotalAmount += payment.Amount

		switch payment.Status {
		case PaymentStatusPending:
			stats.PendingCount++
		case PaymentStatusCompleted:
			stats.CompletedCount++
			stats.CompletedAmount += payment.Amount
		case PaymentStatusFailed:
			stats.FailedCount++
			stats.FailedAmount += payment.Amount
		}
	}

	return stats, nil
}

func (s *paymentService) validateCreatePaymentRequest(tenantID string, req *CreatePaymentRequest) error {
	if tenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}

	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	if req.SourceAccount == "" {
		return fmt.Errorf("source account is required")
	}

	if req.DestinationAccount == "" {
		return fmt.Errorf("destination account is required")
	}

	if req.SourceAccount == req.DestinationAccount {
		return fmt.Errorf("source and destination accounts cannot be the same")
	}

	return nil
}
