package segmentio

import (
	"log/slog"
	"os"
	"time"

	"github.com/nduyhai/xcore/pubsub/kafkit"
)

const BackendSegmentIO kafkit.Backend = "SegmentIO"

type segmentIO struct {
	name kafkit.Backend
}

func newSegmentIO() kafkit.Factory {
	return &segmentIO{name: BackendSegmentIO}
}

func (f *segmentIO) Name() kafkit.Backend {
	return f.name
}

func (f *segmentIO) NewProducer(brokers []string, topic string, opts ...kafkit.ProducerOption) (kafkit.Producer, error) {
	cfg := kafkit.ProducerConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: kafkit.BalancerRoundRobin,
		Async:    false,
		Logger:   slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return newSegmentIOProducer(cfg), nil
}

func (f *segmentIO) NewConsumer(brokers []string, topic string, groupID string, handler kafkit.Handler, opts ...kafkit.ConsumerOption) (kafkit.Consumer, error) {
	cfg := kafkit.ConsumerConfig{
		Brokers:       brokers,
		Topic:         topic,
		GroupID:       groupID,
		MaxConcurrent: 1,
		DLQ: kafkit.DLQConfig{
			Enabled: false,
		},
		Retry: kafkit.RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     10 * time.Second,
			Multiplier:     2,
		},
		CommitInterval:           500 * time.Millisecond,
		HeaderRedactionAllowlist: []string{},
		Logger:                   slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return newSegmentIOConsumer(cfg, handler)
}
