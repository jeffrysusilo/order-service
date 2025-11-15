package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-service/config"
	"order-service/internal/api"
	"order-service/internal/broker"
	"order-service/internal/redisclient"
	"order-service/internal/service"
	"order-service/internal/store"
	"order-service/internal/util"
	"order-service/internal/worker"

	"github.com/gin-gonic/gin"
)

func main() {

	cfg := config.Load()

	if err := util.InitLogger(cfg.Server.Env); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer util.SyncLogger()

	logger := util.GetLogger()
	logger.Info("Starting order service")

	tp, err := util.InitTracer("order-service", cfg.Observ.JaegerEndpoint)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer: %v", err)
		}
	}()

	db, err := store.NewStore(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Database connected")

	redisClient, err := redisclient.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Redis connected")

	producer := broker.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicOrder)
	defer producer.Close()
	log.Println("Kafka producer initialized")

	eventPublisher := broker.NewEventPublisher(producer)

	inventoryClient := service.NewInventoryClient(db, redisClient)
	paymentService := service.NewPaymentService(db, eventPublisher)
	orderService := service.NewOrderService(db, redisClient, eventPublisher, inventoryClient)
	sagaOrchestrator := service.NewSagaOrchestrator(db, inventoryClient, paymentService, eventPublisher)

	ctx := context.Background()
	if err := inventoryClient.SyncInventoryToRedis(ctx); err != nil {
		log.Printf("Failed to sync inventory to Redis: %v", err)
	}

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	orderConsumer := broker.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicOrder, cfg.Kafka.ConsumerGroup)
	orderWorker := worker.NewOrderWorker(orderConsumer, sagaOrchestrator)
	go func() {
		if err := orderWorker.Start(workerCtx); err != nil {
			log.Printf("Order worker error: %v", err)
		}
	}()

	paymentConsumer := broker.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicOrder, "payment-service-group")
	paymentWorker := worker.NewPaymentWorker(paymentConsumer, paymentService)
	go func() {
		if err := paymentWorker.Start(workerCtx); err != nil {
			log.Printf("Payment worker error: %v", err)
		}
	}()

	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	handler := api.NewHandler(orderService)
	handler.SetupRoutes(router)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Starting HTTP server on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	workerCancel()
	orderWorker.Stop()
	paymentWorker.Stop()

	log.Println("Server exited")
}
