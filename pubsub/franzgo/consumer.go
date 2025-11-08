package franzgo

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nduyhai/xcore/pubsub/kafkit"
	"github.com/twmb/franz-go/pkg/kgo"
)

type franzConsumer struct {
	client  *kgo.Client
	cfg     kafkit.ConsumerConfig
	handler kafkit.Handler
	dlq     kafkit.Producer
	retry   kafkit.Producer
	logger  *slog.Logger
}

func newFranzConsumer(cfg kafkit.ConsumerConfig, handler kafkit.Handler) (kafkit.Consumer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ConsumeTopics(cfg.Topic),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
	}
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	var dlqProducer, retryProducer kafkit.Producer
	if cfg.DLQ.Enabled {
		dlqProducer, _ = newFranzProducer(kafkit.ProducerConfig{
			Brokers: cfg.Brokers,
			Topic:   cfg.DLQ.Topic,
			Logger:  cfg.Logger,
		})
	}
	if cfg.Retry.EnableRetryTopic {
		retryProducer, _ = newFranzProducer(kafkit.ProducerConfig{
			Brokers: cfg.Brokers,
			Topic:   cfg.Retry.RetryTopic,
			Logger:  cfg.Logger,
		})
	}

	return &franzConsumer{
		client:  client,
		cfg:     cfg,
		handler: handler,
		dlq:     dlqProducer,
		retry:   retryProducer,
		logger:  cfg.Logger,
	}, nil
}

func (c *franzConsumer) Start(ctx context.Context) error {
	sem := make(chan struct{}, c.cfg.MaxConcurrent)
	ticker := time.NewTicker(c.cfg.CommitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopping")
			return nil
		case <-ticker.C:
			_ = c.client.CommitUncommittedOffsets(ctx)
		default:
			fetches := c.client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, e := range errs {
					if ctx.Err() != nil {
						return nil
					}
					c.logger.Error("poll error", "err", e)
				}
				continue
			}

			iter := fetches.RecordIter()
			for !iter.Done() {
				rec := iter.Next()
				sem <- struct{}{}
				go func(r *kgo.Record) {
					defer func() { <-sem }()
					if err := c.handleWithRetry(ctx, r); err != nil {
						c.logger.Error("handler failed", "topic", r.Topic, "offset", r.Offset, "err", err)
						_ = c.sendToDLQ(ctx, r, err)
					}
					c.client.MarkCommitRecords(r)
				}(rec)
			}
		}
	}
}

func (c *franzConsumer) handleWithRetry(ctx context.Context, rec *kgo.Record) error {
	attempts := 0
	delay := c.cfg.Retry.InitialBackoff

	for {
		attempts++
		err := c.handleMessage(ctx, rec)
		if err == nil {
			return nil
		}

		if c.cfg.Retry.NonRetryable != nil && c.cfg.Retry.NonRetryable(err) {
			c.logger.Warn("non-retryable error", "err", err)
			return err
		}

		if attempts >= c.cfg.Retry.MaxAttempts {
			c.logger.Warn("max retry reached", "err", err)
			if c.retry != nil {
				_ = c.retry.SendWith(ctx, rec.Value, kafkit.WithKey(rec.Key))
			}
			return err
		}

		c.logger.Warn("retrying", "attempt", attempts, "delay", delay, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay = time.Duration(float64(delay) * c.cfg.Retry.Multiplier)
			if delay > c.cfg.Retry.MaxBackoff {
				delay = c.cfg.Retry.MaxBackoff
			}
		}
	}
}

func (c *franzConsumer) handleMessage(ctx context.Context, rec *kgo.Record) error {
	headers := make([]kafkit.Header, len(rec.Headers))
	for i, h := range rec.Headers {
		headers[i] = kafkit.Header{Key: h.Key, Value: h.Value}
	}

	msg := kafkit.ConsumeMessage{
		Topic:     rec.Topic,
		Partition: int(rec.Partition),
		Offset:    rec.Offset,
		Timestamp: rec.Timestamp,
		Key:       rec.Key,
		Value:     rec.Value,
		Headers:   headers,
	}
	return c.handler.Handle(ctx, msg)
}

func (c *franzConsumer) sendToDLQ(ctx context.Context, rec *kgo.Record, cause error) error {
	if c.dlq == nil {
		c.logger.Warn("DLQ disabled; skipping", "err", cause)
		return nil
	}

	redacted := redactHeadersFranz(rec.Headers, c.cfg.HeaderRedactionAllowlist)
	data := map[string]any{
		"topic":     rec.Topic,
		"partition": rec.Partition,
		"offset":    rec.Offset,
		"error":     cause.Error(),
		"headers":   redacted,
		"payload":   string(rec.Value),
		"timestamp": rec.Timestamp,
	}
	bytes, _ := json.Marshal(data)
	return c.dlq.SendWith(ctx, bytes, kafkit.WithKey(rec.Key))
}

func redactHeadersFranz(headers []kgo.RecordHeader, allowlist []string) map[string]string {
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

func (c *franzConsumer) Close(ctx context.Context) error {
	c.client.Close()
	return nil
}
