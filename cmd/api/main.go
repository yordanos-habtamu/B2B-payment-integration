package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	customMiddleware "github.com/yordanos-habtamu/b2b-payments/internal/server/middleware"
	"github.com/yordanos-habtamu/b2b-payments/internal/service"
	"github.com/yordanos-habtamu/b2b-payments/internal/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/yordanos-habtamu/b2b-payments/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	e := echo.New()

	// Basic middleware
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	// Initialize services
	paymentService := service.NewPaymentService()
	paymentHandler := handler.NewPaymentHandler(paymentService)

	// Health check (no auth required)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Protected API group — all routes under /api require tenant auth
	api := e.Group("/api/v1")
	api.Use(customMiddleware.TenantExtraction())
	
	// Initialize OPA middleware
	opaMiddleware, err := customMiddleware.NewOPAMiddleware()
	if err != nil {
		log.Fatalf("Failed to initialize OPA middleware: %v", err)
	}
	
	// <-- Zero Trust tenant scoping
	// Initialize Redis-backed idempotency
	idempotency, err := customMiddleware.NewIdempotency(cfg.RedisURL, cfg.IdempotencyTTL)
	if err != nil {
		log.Fatalf("Failed to initialize idempotency: %v", err)
	}

	api.Use(customMiddleware.TenantExtraction())
	api.Use(opaMiddleware.Authorize()) // <-- OPA policy-based authorization
	api.Use(idempotency.Idempotent()) // <-- Idempotency protection

	// Payment routes
	payments := api.Group("/payments")
	payments.GET("", paymentHandler.ListPayments)
	payments.POST("", paymentHandler.CreatePayment)
	payments.GET("/stats", paymentHandler.GetPaymentStats)
	payments.GET("/:id", paymentHandler.GetPayment)
	payments.PUT("/:id", paymentHandler.UpdatePayment)
	payments.POST("/:id/process", paymentHandler.ProcessPayment)
	payments.POST("/:id/cancel", paymentHandler.CancelPayment)

	// Legacy endpoint for backward compatibility
	api.GET("/payments", func(c echo.Context) error {
		tenantID, err := customMiddleware.GetTenantID(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get tenant")
		}

		return c.JSON(http.StatusOK, map[string]any{
			"message":   "Welcome to B2B Payments API",
			"tenant_id": tenantID,
			"hint":      "You're securely authenticated via mTLS + tenant cert!",
		})
	})

	// Optional: add a simple endpoint to echo cert details (useful for debugging)
	api.GET("/whoami", func(c echo.Context) error {
		tlsState := c.Request().TLS
		if tlsState == nil || len(tlsState.PeerCertificates) == 0 {
			return c.JSON(http.StatusBadRequest, "no client cert")
		}
		cert := tlsState.PeerCertificates[0]

		tenantID, _ := customMiddleware.GetTenantID(c)

		return c.JSON(http.StatusOK, map[string]any{
			"tenant_id":       tenantID,
			"cert_cn":         cert.Subject.CommonName,
			"cert_serial":     cert.SerialNumber.String(),
			"issuer":          cert.Issuer.CommonName,
			"not_before":      cert.NotBefore,
			"not_after":       cert.NotAfter,
			"dns_names":       cert.DNSNames,
			"verified_chains": len(tlsState.VerifiedChains) > 0,
		})
	})

	// Load server cert/key
	serverCert, err := tls.LoadX509KeyPair(cfg.ServerCert, cfg.ServerKey)
	if err != nil {
		log.Fatalf("Failed to load server cert/key: %v", err)
	}

	// Load CA for client verification
	caCertPEM, err := ioutil.ReadFile(cfg.CAFile)
	if err != nil {
		log.Fatalf("Failed to read CA file: %v", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		log.Fatalf("Failed to append CA cert")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert, // Zero Trust core
		MinVersion:   tls.VersionTLS13,
	}

	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", cfg.Port),
		Handler:   e,
		TLSConfig: tlsConfig,
	}

	go func() {
		log.Printf("Starting mTLS server on :%d", cfg.Port)
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}
