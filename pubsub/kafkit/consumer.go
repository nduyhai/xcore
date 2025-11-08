package kafkit

import (
	"context"
	"log/slog"
	"time"
)

type HandlerFunc func(ctx context.Context, msg ConsumeMessage) error

func (f HandlerFunc) Handle(ctx context.Context, msg ConsumeMessage) error { return f(ctx, msg) }

type Handler interface {
	Handle(ctx context.Context, msg ConsumeMessage) error
}

type Consumer interface {
	Start(ctx context.Context) error
	Close(ctx context.Context) error
}

type ConsumerConfig struct {
	Brokers []string
	Topic   string
	GroupID string

	Retry                    RetryPolicy
	DLQ                      DLQConfig
	MaxConcurrent            int
	CommitInterval           time.Duration
	HeaderRedactionAllowlist []string
	Logger                   *slog.Logger
}

type RetryPolicy struct {
	MaxAttempts      int
	InitialBackoff   time.Duration
	MaxBackoff       time.Duration
	Multiplier       float64
	NonRetryable     func(error) bool
	RetryTopic       string
	EnableRetryTopic bool
}

type DLQConfig struct {
	Enabled         bool
	Topic           string
	KeyFromOriginal bool
}

type ConsumerOption func(*ConsumerConfig)

func WithDLQ(topic string, keyFromOriginal bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.DLQ = DLQConfig{
			Enabled:         true,
			Topic:           topic,
			KeyFromOriginal: keyFromOriginal,
		}
	}
}

func WithoutDLQ() ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.DLQ = DLQConfig{}
	}
}

func WithRetryPolicy(maxAttempts int, initial, maxBackoff time.Duration, multiplier float64) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Retry = RetryPolicy{
			MaxAttempts:    maxAttempts,
			InitialBackoff: initial,
			MaxBackoff:     maxBackoff,
			Multiplier:     multiplier,
		}
	}
}

func WithNonRetryable(fn func(error) bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Retry.NonRetryable = fn
	}
}

func WithRetryTopic(topic string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Retry.RetryTopic = topic
		cfg.Retry.EnableRetryTopic = true
	}
}

func WithMaxConcurrent(n int) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		if n <= 0 {
			n = 1
		}
		cfg.MaxConcurrent = n
	}
}

func WithCommitInterval(interval time.Duration) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.CommitInterval = interval
	}
}

func WithHeaderAllowlist(keys ...string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.HeaderRedactionAllowlist = append(cfg.HeaderRedactionAllowlist, keys...)
	}
}
