package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	Observ   ObservabilityConfig
	Business BusinessConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers       []string
	TopicOrder    string
	ConsumerGroup string
}

type ObservabilityConfig struct {
	JaegerEndpoint string
	PrometheusPort string
}

type BusinessConfig struct {
	OrderTimeoutSeconds   int
	PaymentTimeoutSeconds int
}

func Load() *Config {
	_ = godotenv.Load()

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	orderTimeout, _ := strconv.Atoi(getEnv("ORDER_TIMEOUT_SECONDS", "300"))
	paymentTimeout, _ := strconv.Atoi(getEnv("PAYMENT_TIMEOUT_SECONDS", "60"))

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://app:secret@localhost:5432/app?sslmode=disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Kafka: KafkaConfig{
			Brokers:       strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
			TopicOrder:    getEnv("KAFKA_TOPIC_ORDER_EVENTS", "order-events"),
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "order-service-group"),
		},
		Observ: ObservabilityConfig{
			JaegerEndpoint: getEnv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
			PrometheusPort: getEnv("PROMETHEUS_PORT", "9090"),
		},
		Business: BusinessConfig{
			OrderTimeoutSeconds:   orderTimeout,
			PaymentTimeoutSeconds: paymentTimeout,
		},
	}

	log.Printf("Config loaded: env=%s, port=%s", cfg.Server.Env, cfg.Server.Port)
	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
