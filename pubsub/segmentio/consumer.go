package segmentio

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/nduyhai/xcore/pubsub/kafkit"
	"github.com/segmentio/kafka-go"
)

const defaultProducerSendTimeout = 5 * time.Second

type segmentIOConsumer struct {
	reader  segmentioReader
	dlq     kafkit.Producer
	retry   kafkit.Producer
	cfg     kafkit.ConsumerConfig
	handler kafkit.Handler
	logger  *slog.Logger
	wg      sync.WaitGroup
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
	go func() {
		err := s.start(ctx)
		if err != nil {
			s.logger.Error("consumer failed", "err", err)
		}
	}()
	return nil
}

func (s *segmentIOConsumer) start(ctx context.Context) error {
	maxWorkers := s.cfg.MaxConcurrent
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	msgBuf := maxWorkers * 2
	messages := make(chan kafka.Message, msgBuf)
	commitChan := make(chan kafka.Message, msgBuf)

	// start commit worker (track with s.wg)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// commit worker: commit each message as it arrives
		for m := range commitChan {
			if err := s.reader.CommitMessages(ctx, m); err != nil {
				s.logger.Error("commit failed", "err", err)
			}
		}
	}()

	// start workers (track with local workerWG so we can deterministically close commitChan)
	var workerWG sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		workerWG.Add(1)
		go func(workerID int) {
			defer workerWG.Done()
			for m := range messages {
				if err := s.handleWithRetry(ctx, m); err != nil {
					s.logger.Error("message failed", "topic", m.Topic, "partition", m.Partition, "offset", m.Offset, "err", err)
					_ = s.sendToDLQ(ctx, m, err)
				}
				// forward to commit a channel regardless of error to keep previous semantics
				select {
				case commitChan <- m:
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	ticker := time.NewTicker(s.cfg.CommitInterval)
	defer ticker.Stop()

fetchLoop:
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("consumer shutting down - stopping fetch and waiting for in-flight handlers")
			// stop fetching and break to shutdown: close messages then wait for workers
			break fetchLoop

		case <-ticker.C:
			// keep-alive event; nothing to do here
			continue

		default:
			msg, err := s.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					// context canceled; break fetch loop and shutdown
					s.logger.Info("consumer context cancelled - stopping fetch and waiting for in-flight handlers")
					break fetchLoop
				}
				s.logger.Error("fetch error", "err", err)
				continue
			}

			select {
			case messages <- msg:
			case <-ctx.Done():
				// if context cancelled while trying to push, stop
				break fetchLoop
			}
		}
	}

	// shutdown sequence: stop accepting messages, wait for workers to finish, then close commit channel and wait for commit worker
	close(messages)
	// wait for workers to finish forwarding to commitChan
	workerWG.Wait()
	// all workers done; safe to close commitChan to allow commit worker to finish
	close(commitChan)

	// wait for the commit worker to finish
	s.wg.Wait()
	return nil
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
				// use a bounded timeout for retry sends to avoid blocking indefinitely
				timeout := s.cfg.ProducerSendTimeout
				if timeout <= 0 {
					timeout = defaultProducerSendTimeout
				}
				sendCtx, cancel := context.WithTimeout(ctx, timeout)
				_ = s.retry.SendWith(sendCtx, msg.Value, kafkit.WithKey(msg.Key))
				cancel()
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

	data, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("failed to marshal DLQ payload", "err", err)
		return err
	}

	// use a bounded timeout for DLQ sends to avoid blocking handlers indefinitely
	timeout := s.cfg.ProducerSendTimeout
	if timeout <= 0 {
		timeout = defaultProducerSendTimeout
	}

	sendCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = s.dlq.SendWith(sendCtx, data, kafkit.WithKey(msg.Key))
	if err != nil {
		s.logger.Error("failed to send to DLQ", "err", err)
	}
	return err
}

func (s *segmentIOConsumer) Close(ctx context.Context) error {
	var readerErr error
	if s.reader != nil {
		readerErr = s.reader.Close()
	}

	// wait for in-flight handlers to finish, respecting ctx
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// after all workers/commit worker finished, close producers
		var prodErr error
		if s.dlq != nil {
			if err := s.dlq.Close(ctx); err != nil {
				prodErr = err
			}
		}
		if s.retry != nil {
			if err := s.retry.Close(ctx); err != nil {
				if prodErr == nil {
					prodErr = err
				}
			}
		}
		// prefer reader error if present, otherwise producer close error
		if readerErr != nil {
			return readerErr
		}
		return prodErr
	case <-ctx.Done():
		// prefer returning ctx.Err() so callers know we timed out/cancelled while waiting
		if readerErr != nil {
			return readerErr
		}
		return ctx.Err()
	}
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
