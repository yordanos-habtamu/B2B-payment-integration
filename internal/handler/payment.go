package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/yordanos-habtamu/b2b-payments/internal/server/middleware"
	"github.com/yordanos-habtamu/b2b-payments/internal/service"
)

type PaymentHandler struct {
	paymentService service.PaymentService
}

func NewPaymentHandler(paymentService service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// CreatePayment creates a new payment
// @Summary Create a new payment
// @Description Creates a new payment for the authenticated tenant
// @Tags payments
// @Accept json
// @Produce json
// @Param payment body service.CreatePaymentRequest true "Payment data"
// @Success 201 {object} service.Payment
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payments [post]
// @Security BearerAuth
func (h *PaymentHandler) CreatePayment(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req service.CreatePaymentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	payment, err := h.paymentService.CreatePayment(c.Request().Context(), tenantID, &req)
	if err != nil {
		c.Logger().Error("Failed to create payment", "error", err, "tenant", tenantID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create payment")
	}

	return c.JSON(http.StatusCreated, payment)
}

// GetPayment retrieves a specific payment
// @Summary Get a payment by ID
// @Description Retrieves a specific payment for the authenticated tenant
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} service.Payment
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payments/{id} [get]
// @Security BearerAuth
func (h *PaymentHandler) GetPayment(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	paymentID := c.Param("id")
	if paymentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "payment ID is required")
	}

	payment, err := h.paymentService.GetPayment(c.Request().Context(), tenantID, paymentID)
	if err != nil {
		if err.Error() == "payment not found" {
			return echo.NewHTTPError(http.StatusNotFound, "payment not found")
		}
		if err.Error() == "access denied" {
			return echo.NewHTTPError(http.StatusForbidden, "access denied")
		}
		c.Logger().Error("Failed to get payment", "error", err, "tenant", tenantID, "payment", paymentID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve payment")
	}

	return c.JSON(http.StatusOK, payment)
}

// UpdatePayment updates an existing payment
// @Summary Update a payment
// @Description Updates an existing payment for the authenticated tenant
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Param payment body service.UpdatePaymentRequest true "Payment update data"
// @Success 200 {object} service.Payment
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payments/{id} [put]
// @Security BearerAuth
func (h *PaymentHandler) UpdatePayment(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	paymentID := c.Param("id")
	if paymentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "payment ID is required")
	}

	var req service.UpdatePaymentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	payment, err := h.paymentService.UpdatePayment(c.Request().Context(), tenantID, paymentID, &req)
	if err != nil {
		if err.Error() == "payment not found" {
			return echo.NewHTTPError(http.StatusNotFound, "payment not found")
		}
		if err.Error() == "access denied" {
			return echo.NewHTTPError(http.StatusForbidden, "access denied")
		}
		c.Logger().Error("Failed to update payment", "error", err, "tenant", tenantID, "payment", paymentID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update payment")
	}

	return c.JSON(http.StatusOK, payment)
}

// ListPayments retrieves a list of payments with filtering and pagination
// @Summary List payments
// @Description Retrieves a list of payments for the authenticated tenant with filtering and pagination
// @Tags payments
// @Accept json
// @Produce json
// @Param status query string false "Payment status filter" Enums(pending,processing,completed,failed,cancelled)
// @Param type query string false "Payment type filter" Enums(credit,debit)
// @Param currency query string false "Currency filter" Enums(USD,EUR,GBP)
// @Param min_amount query number false "Minimum amount filter"
// @Param max_amount query number false "Maximum amount filter"
// @Param from_date query string false "From date filter (RFC3339 format)"
// @Param to_date query string false "To date filter (RFC3339 format)"
// @Param limit query int false "Limit number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} map[string]interface{} "Payments list with pagination info"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payments [get]
// @Security BearerAuth
func (h *PaymentHandler) ListPayments(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	filter := &service.PaymentFilter{}

	// Parse query parameters
	if statusStr := c.QueryParam("status"); statusStr != "" {
		status := service.PaymentStatus(statusStr)
		filter.Status = &status
	}

	if typeStr := c.QueryParam("type"); typeStr != "" {
		paymentType := service.PaymentType(typeStr)
		filter.Type = &paymentType
	}

	if currencyStr := c.QueryParam("currency"); currencyStr != "" {
		currency := service.Currency(currencyStr)
		filter.Currency = &currency
	}

	if minAmountStr := c.QueryParam("min_amount"); minAmountStr != "" {
		if minAmount, err := strconv.ParseFloat(minAmountStr, 64); err == nil {
			filter.MinAmount = &minAmount
		}
	}

	if maxAmountStr := c.QueryParam("max_amount"); maxAmountStr != "" {
		if maxAmount, err := strconv.ParseFloat(maxAmountStr, 64); err == nil {
			filter.MaxAmount = &maxAmount
		}
	}

	if fromDateStr := c.QueryParam("from_date"); fromDateStr != "" {
		if fromDate, err := time.Parse(time.RFC3339, fromDateStr); err == nil {
			filter.FromDate = &fromDate
		}
	}

	if toDateStr := c.QueryParam("to_date"); toDateStr != "" {
		if toDate, err := time.Parse(time.RFC3339, toDateStr); err == nil {
			filter.ToDate = &toDate
		}
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Set default pagination
	if filter.Limit == 0 {
		filter.Limit = 50 // Default limit
	}

	payments, total, err := h.paymentService.ListPayments(c.Request().Context(), tenantID, filter)
	if err != nil {
		c.Logger().Error("Failed to list payments", "error", err, "tenant", tenantID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve payments")
	}

	response := map[string]interface{}{
		"payments": payments,
		"total":    total,
		"limit":    filter.Limit,
		"offset":   filter.Offset,
	}

	return c.JSON(http.StatusOK, response)
}

// ProcessPayment processes a payment
// @Summary Process a payment
// @Description Processes a pending payment for the authenticated tenant
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payments/{id}/process [post]
// @Security BearerAuth
func (h *PaymentHandler) ProcessPayment(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	paymentID := c.Param("id")
	if paymentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "payment ID is required")
	}

	err = h.paymentService.ProcessPayment(c.Request().Context(), tenantID, paymentID)
	if err != nil {
		if err.Error() == "payment not found" {
			return echo.NewHTTPError(http.StatusNotFound, "payment not found")
		}
		if err.Error() == "access denied" {
			return echo.NewHTTPError(http.StatusForbidden, "access denied")
		}
		c.Logger().Error("Failed to process payment", "error", err, "tenant", tenantID, "payment", paymentID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to process payment")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "payment processed successfully"})
}

// CancelPayment cancels a payment
// @Summary Cancel a payment
// @Description Cancels a pending or processing payment for the authenticated tenant
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payments/{id}/cancel [post]
// @Security BearerAuth
func (h *PaymentHandler) CancelPayment(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	paymentID := c.Param("id")
	if paymentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "payment ID is required")
	}

	err = h.paymentService.CancelPayment(c.Request().Context(), tenantID, paymentID)
	if err != nil {
		if err.Error() == "payment not found" {
			return echo.NewHTTPError(http.StatusNotFound, "payment not found")
		}
		if err.Error() == "access denied" {
			return echo.NewHTTPError(http.StatusForbidden, "access denied")
		}
		c.Logger().Error("Failed to cancel payment", "error", err, "tenant", tenantID, "payment", paymentID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to cancel payment")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "payment cancelled successfully"})
}

// GetPaymentStats retrieves payment statistics
// @Summary Get payment statistics
// @Description Retrieves payment statistics for the authenticated tenant
// @Tags payments
// @Accept json
// @Produce json
// @Success 200 {object} service.PaymentStats
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payments/stats [get]
// @Security BearerAuth
func (h *PaymentHandler) GetPaymentStats(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	stats, err := h.paymentService.GetPaymentStats(c.Request().Context(), tenantID)
	if err != nil {
		c.Logger().Error("Failed to get payment stats", "error", err, "tenant", tenantID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve payment statistics")
	}

	return c.JSON(http.StatusOK, stats)
}
