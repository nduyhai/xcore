package segmentio

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nduyhai/xcore/pubsub/kafkit"
	"github.com/segmentio/kafka-go"
)

type segmentIOConsumer struct {
	reader  segmentioReader
	dlq     kafkit.Producer
	retry   kafkit.Producer
	cfg     kafkit.ConsumerConfig
	handler kafkit.Handler
	logger  *slog.Logger
}

type segmentioReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

func newSegmentIOConsumer(cfg kafkit.ConsumerConfig, handler kafkit.Handler) (kafkit.Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		GroupID: cfg.GroupID,
		Topic:   cfg.Topic,
	})

	var dlqProducer, retryProducer kafkit.Producer

	if cfg.DLQ.Enabled {
		dlqProducer = newSegmentIOProducer(kafkit.ProducerConfig{
			Brokers: cfg.Brokers,
			Topic:   cfg.DLQ.Topic,
			Logger:  cfg.Logger,
		})
	}

	if cfg.Retry.EnableRetryTopic {
		retryProducer = newSegmentIOProducer(kafkit.ProducerConfig{
			Brokers: cfg.Brokers,
			Topic:   cfg.Retry.RetryTopic,
			Logger:  cfg.Logger,
		})
	}

	return &segmentIOConsumer{
		reader:  reader,
		cfg:     cfg,
		handler: handler,
		logger:  cfg.Logger,
		dlq:     dlqProducer,
		retry:   retryProducer,
	}, nil
}

func (s *segmentIOConsumer) Start(ctx context.Context) error {
	sem := make(chan struct{}, s.cfg.MaxConcurrent)
	ticker := time.NewTicker(s.cfg.CommitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("consumer shutting down")
			return nil

		case <-ticker.C:
			// periodic offset commit can be implemented if desired
			continue

		default:
			msg, err := s.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				s.logger.Error("fetch error", "err", err)
				continue
			}

			sem <- struct{}{}
			go func(m kafka.Message) {
				defer func() { <-sem }()
				if err := s.handleWithRetry(ctx, m); err != nil {
					s.logger.Error("message failed", "topic", m.Topic, "partition", m.Partition, "offset", m.Offset, "err", err)
					_ = s.sendToDLQ(ctx, m, err)
				}
				if err := s.reader.CommitMessages(ctx, m); err != nil {
					s.logger.Error("commit failed", "err", err)
				}
			}(msg)
		}
	}
}

func (s *segmentIOConsumer) handleWithRetry(ctx context.Context, msg kafka.Message) error {
	attempts := 0
	delay := s.cfg.Retry.InitialBackoff

	for {
		attempts++
		err := s.handleMessage(ctx, msg)
		if err == nil {
			return nil
		}

		if s.cfg.Retry.NonRetryable != nil && s.cfg.Retry.NonRetryable(err) {
			s.logger.Warn("non-retryable error", "err", err)
			return err
		}

		if attempts >= s.cfg.Retry.MaxAttempts {
			s.logger.Warn("max attempts reached, send to retry topic", "err", err)
			if s.retry != nil {
				_ = s.retry.SendWith(ctx, msg.Value, kafkit.WithKey(msg.Key))
			}
			return err
		}

		s.logger.Warn("retrying", "attempt", attempts, "delay", delay, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay = time.Duration(float64(delay) * s.cfg.Retry.Multiplier)
			if delay > s.cfg.Retry.MaxBackoff {
				delay = s.cfg.Retry.MaxBackoff
			}
		}
	}
}

func (s *segmentIOConsumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	headers := make([]kafkit.Header, len(msg.Headers))
	for i, h := range msg.Headers {
		headers[i] = kafkit.Header{Key: h.Key, Value: h.Value}
	}
	return s.handler.Handle(ctx, kafkit.ConsumeMessage{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Timestamp: msg.Time,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   headers,
	})
}

func (s *segmentIOConsumer) sendToDLQ(ctx context.Context, msg kafka.Message, cause error) error {
	if s.dlq == nil {
		s.logger.Warn("DLQ disabled; skipping", "err", cause)
		return nil
	}

	redacted := redactSegmentIOHeaders(msg.Headers, s.cfg.HeaderRedactionAllowlist)
	payload := map[string]any{
		"topic":     msg.Topic,
		"partition": msg.Partition,
		"offset":    msg.Offset,
		"error":     cause.Error(),
		"headers":   redacted,
		"payload":   string(msg.Value),
		"timestamp": msg.Time,
	}

	data, _ := json.Marshal(payload)
	err := s.dlq.SendWith(ctx, data, kafkit.WithKey(msg.Key))
	if err != nil {
		s.logger.Error("failed to send to DLQ", "err", err)
	}
	return err
}

func (s *segmentIOConsumer) Close(ctx context.Context) error {
	if s.reader != nil {
		return s.reader.Close()
	}
	return nil
}

func redactSegmentIOHeaders(headers []kafka.Header, allowlist []string) map[string]string {
	allowed := make(map[string]struct{}, len(allowlist))
	for _, k := range allowlist {
		allowed[k] = struct{}{}
	}
	out := make(map[string]string)
	for _, h := range headers {
		if _, ok := allowed[h.Key]; ok {
			out[h.Key] = string(h.Value)
		} else {
			out[h.Key] = "[REDACTED]"
		}
	}
	return out
}
