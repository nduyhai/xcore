package franzgo

import (
	"context"
	"log/slog"

	"github.com/nduyhai/xcore/pubsub/kafkit"
	"github.com/twmb/franz-go/pkg/kgo"
)

type franzProducer struct {
	client *kgo.Client
	cfg    kafkit.ProducerConfig
	logger *slog.Logger
}

func newFranzProducer(cfg kafkit.ProducerConfig) (kafkit.Producer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.DefaultProduceTopic(cfg.Topic),
		chooseFranzGoBalancer(cfg.Balancer),
	}

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &franzProducer{client: cl, cfg: cfg, logger: cfg.Logger}, nil
}

func chooseFranzGoBalancer(b kafkit.BalancerStrategy) kgo.Opt {
	switch b {
	case kafkit.BalancerHash:
		return kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)) //TODO: verify if this is the correct one for hash
	case kafkit.BalancerSticky:
		return kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil))
	case kafkit.BalancerManual:
		return kgo.RecordPartitioner(kgo.ManualPartitioner())
	default:
		return kgo.RecordPartitioner(kgo.RoundRobinPartitioner())
	}
}

func (f *franzProducer) SendWith(ctx context.Context, value []byte, opts ...kafkit.ProduceOption) error {
	msg := kafkit.ProduceMessage{Value: value}
	for _, o := range opts {
		o(&msg)
	}

	hdrs := make([]kgo.RecordHeader, len(msg.Headers))
	for i, h := range msg.Headers {
		hdrs[i] = kgo.RecordHeader{Key: h.Key, Value: h.Value}
	}

	rec := &kgo.Record{
		Topic:   f.cfg.Topic,
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: hdrs,
	}

	f.client.Produce(ctx, rec, func(_ *kgo.Record, err error) {
		if err != nil {
			f.logger.Error("produce failed", "topic", f.cfg.Topic, "err", err)
		}
	})
	return nil
}

func (f *franzProducer) Close(ctx context.Context) error {
	f.client.Close()
	return nil
}
