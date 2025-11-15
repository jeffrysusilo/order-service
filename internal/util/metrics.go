package util

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	OrdersCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "orders_created_total",
		Help: "Total number of orders created",
	})

	OrdersReservedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "orders_reserved_total",
		Help: "Total number of orders with inventory reserved",
	})

	OrdersPaidTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "orders_paid_total",
		Help: "Total number of orders successfully paid",
	})

	OrdersFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "orders_failed_total",
		Help: "Total number of failed orders",
	}, []string{"reason"})

	OrdersCancelledTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "orders_cancelled_total",
		Help: "Total number of cancelled orders",
	})

	InventoryReserveLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "inventory_reserve_latency_seconds",
		Help:    "Latency of inventory reservation operations",
		Buckets: prometheus.DefBuckets,
	})

	InventoryReservationsFailed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "inventory_reservations_failed_total",
		Help: "Total number of failed inventory reservations",
	}, []string{"reason"})

	PaymentAttemptsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "payment_attempts_total",
		Help: "Total number of payment attempts",
	})

	PaymentSuccessTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "payment_success_total",
		Help: "Total number of successful payments",
	})

	PaymentFailedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "payment_failed_total",
		Help: "Total number of failed payments",
	})

	PaymentProcessingLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "payment_processing_latency_seconds",
		Help:    "Latency of payment processing",
		Buckets: prometheus.DefBuckets,
	})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})
)
