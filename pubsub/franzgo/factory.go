package franzgo

import (
	"log/slog"
	"os"
	"time"

	"github.com/nduyhai/xcore/pubsub/kafkit"
)

const BackendFranzGO kafkit.Backend = "FranzGo"

type franzGo struct {
	name kafkit.Backend
}

func newFranzGo() kafkit.Factory {
	return &franzGo{name: BackendFranzGO}
}

func (f *franzGo) Name() kafkit.Backend {
	return f.name
}

func (f *franzGo) NewProducer(brokers []string, topic string, opts ...kafkit.ProducerOption) (kafkit.Producer, error) {
	cfg := kafkit.ProducerConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: kafkit.BalancerRoundRobin,
		Async:    true,
		Logger:   slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return newFranzProducer(cfg)
}

func (f *franzGo) NewConsumer(brokers []string, topic string, groupID string, handler kafkit.Handler, opts ...kafkit.ConsumerOption) (kafkit.Consumer, error) {
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

	return newFranzConsumer(cfg, handler)
}
