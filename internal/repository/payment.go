package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yordanos-habtamu/b2b-payments/internal/service"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *service.Payment) error
	GetByID(ctx context.Context, tenantID, paymentID string) (*service.Payment, error)
	Update(ctx context.Context, payment *service.Payment) error
	List(ctx context.Context, tenantID string, filter *service.PaymentFilter) ([]*service.Payment, int64, error)
	GetStats(ctx context.Context, tenantID string) (*service.PaymentStats, error)
	Delete(ctx context.Context, tenantID, paymentID string) error
}

type paymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) PaymentRepository {
	return &paymentRepository{
		db: db,
	}
}

func (r *paymentRepository) Create(ctx context.Context, payment *service.Payment) error {
	query := `
		INSERT INTO payments (
			id, tenant_id, amount, currency, type, status, description,
			reference, source_account, destination_account, metadata,
			created_at, updated_at, processed_at, completed_at, failed_at, failure_reason
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)`

	_, err := r.db.Exec(ctx, query,
		payment.ID,
		payment.TenantID,
		payment.Amount,
		payment.Currency,
		payment.Type,
		payment.Status,
		payment.Description,
		payment.Reference,
		payment.SourceAccount,
		payment.DestinationAccount,
		payment.Metadata,
		payment.CreatedAt,
		payment.UpdatedAt,
		payment.ProcessedAt,
		payment.CompletedAt,
		payment.FailedAt,
		payment.FailureReason,
	)

	return err
}

func (r *paymentRepository) GetByID(ctx context.Context, tenantID, paymentID string) (*service.Payment, error) {
	query := `
		SELECT id, tenant_id, amount, currency, type, status, description,
			   reference, source_account, destination_account, metadata,
			   created_at, updated_at, processed_at, completed_at, failed_at, failure_reason
		FROM payments
		WHERE id = $1 AND tenant_id = $2`

	var payment service.Payment
	var metadata sql.Null[map[string]interface{}]

	err := r.db.QueryRow(ctx, query, paymentID, tenantID).Scan(
		&payment.ID,
		&payment.TenantID,
		&payment.Amount,
		&payment.Currency,
		&payment.Type,
		&payment.Status,
		&payment.Description,
		&payment.Reference,
		&payment.SourceAccount,
		&payment.DestinationAccount,
		&metadata,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.ProcessedAt,
		&payment.CompletedAt,
		&payment.FailedAt,
		&payment.FailureReason,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, err
	}

	if metadata.Valid {
		payment.Metadata = metadata.V
	}

	return &payment, nil
}

func (r *paymentRepository) Update(ctx context.Context, payment *service.Payment) error {
	query := `
		UPDATE payments SET
			amount = $3,
			currency = $4,
			type = $5,
			status = $6,
			description = $7,
			reference = $8,
			source_account = $9,
			destination_account = $10,
			metadata = $11,
			updated_at = $12,
			processed_at = $13,
			completed_at = $14,
			failed_at = $15,
			failure_reason = $16
		WHERE id = $1 AND tenant_id = $2`

	_, err := r.db.Exec(ctx, query,
		payment.ID,
		payment.TenantID,
		payment.Amount,
		payment.Currency,
		payment.Type,
		payment.Status,
		payment.Description,
		payment.Reference,
		payment.SourceAccount,
		payment.DestinationAccount,
		payment.Metadata,
		payment.UpdatedAt,
		payment.ProcessedAt,
		payment.CompletedAt,
		payment.FailedAt,
		payment.FailureReason,
	)

	return err
}

func (r *paymentRepository) List(ctx context.Context, tenantID string, filter *service.PaymentFilter) ([]*service.Payment, int64, error) {
	// Build WHERE clause
	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argIndex := 2

	if filter != nil {
		if filter.Status != nil {
			whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
			args = append(args, *filter.Status)
			argIndex++
		}
		if filter.Type != nil {
			whereClause += fmt.Sprintf(" AND type = $%d", argIndex)
			args = append(args, *filter.Type)
			argIndex++
		}
		if filter.Currency != nil {
			whereClause += fmt.Sprintf(" AND currency = $%d", argIndex)
			args = append(args, *filter.Currency)
			argIndex++
		}
		if filter.MinAmount != nil {
			whereClause += fmt.Sprintf(" AND amount >= $%d", argIndex)
			args = append(args, *filter.MinAmount)
			argIndex++
		}
		if filter.MaxAmount != nil {
			whereClause += fmt.Sprintf(" AND amount <= $%d", argIndex)
			args = append(args, *filter.MaxAmount)
			argIndex++
		}
		if filter.FromDate != nil {
			whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
			args = append(args, *filter.FromDate)
			argIndex++
		}
		if filter.ToDate != nil {
			whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
			args = append(args, *filter.ToDate)
			argIndex++
		}
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM payments " + whereClause
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Build ORDER BY and LIMIT clauses
	orderClause := "ORDER BY created_at DESC"
	limitClause := ""
	
	if filter != nil && filter.Limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
		
		if filter.Offset > 0 {
			limitClause += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	// Main query
	query := `
		SELECT id, tenant_id, amount, currency, type, status, description,
			   reference, source_account, destination_account, metadata,
			   created_at, updated_at, processed_at, completed_at, failed_at, failure_reason
		FROM payments ` + whereClause + " " + orderClause + " " + limitClause

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []*service.Payment
	for rows.Next() {
		var payment service.Payment
		var metadata sql.Null[map[string]interface{}]

		err := rows.Scan(
			&payment.ID,
			&payment.TenantID,
			&payment.Amount,
			&payment.Currency,
			&payment.Type,
			&payment.Status,
			&payment.Description,
			&payment.Reference,
			&payment.SourceAccount,
			&payment.DestinationAccount,
			&metadata,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&payment.ProcessedAt,
			&payment.CompletedAt,
			&payment.FailedAt,
			&payment.FailureReason,
		)

		if err != nil {
			return nil, 0, err
		}

		if metadata.Valid {
			payment.Metadata = metadata.V
		}

		payments = append(payments, &payment)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

func (r *paymentRepository) GetStats(ctx context.Context, tenantID string) (*service.PaymentStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_count,
			COALESCE(SUM(amount), 0) as total_amount,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_count,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_count,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN amount ELSE 0 END), 0) as completed_amount,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN amount ELSE 0 END), 0) as failed_amount
		FROM payments
		WHERE tenant_id = $1`

	var stats service.PaymentStats
	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&stats.TotalCount,
		&stats.TotalAmount,
		&stats.PendingCount,
		&stats.CompletedCount,
		&stats.FailedCount,
		&stats.CompletedAmount,
		&stats.FailedAmount,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *paymentRepository) Delete(ctx context.Context, tenantID, paymentID string) error {
	query := "DELETE FROM payments WHERE id = $1 AND tenant_id = $2"
	
	result, err := r.db.Exec(ctx, query, paymentID, tenantID)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("payment not found")
	}

	return nil
}
