package kafkit

import (
	"context"
	"log/slog"
)

type Producer interface {
	SendWith(ctx context.Context, value []byte, opts ...ProduceOption) error
	Close(ctx context.Context) error
}

type ProduceOption func(*ProduceMessage)

func WithKey(key []byte) ProduceOption {
	return func(msg *ProduceMessage) { msg.Key = key }
}

func WithHeader(key string, value []byte) ProduceOption {
	return func(msg *ProduceMessage) {
		msg.Headers = append(msg.Headers, Header{Key: key, Value: value})
	}
}

func WithHeaders(headers map[string][]byte) ProduceOption {
	return func(msg *ProduceMessage) {
		for k, v := range headers {
			msg.Headers = append(msg.Headers, Header{Key: k, Value: v})
		}
	}
}

type ProducerConfig struct {
	Brokers []string
	Topic   string

	Balancer BalancerStrategy
	Async    bool
	Logger   *slog.Logger
}

type ProducerOption func(*ProducerConfig)

func WithBalancer(b BalancerStrategy) ProducerOption {
	return func(cfg *ProducerConfig) {
		cfg.Balancer = b
	}
}

func WithAsync(enabled bool) ProducerOption {
	return func(cfg *ProducerConfig) {
		cfg.Async = enabled
	}
}
