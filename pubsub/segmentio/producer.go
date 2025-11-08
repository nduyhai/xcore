package segmentio

import (
	"context"
	"log/slog"

	"github.com/nduyhai/xcore/pubsub/kafkit"
	"github.com/segmentio/kafka-go"
)

type segmentIOProducer struct {
	writer *kafka.Writer
	cfg    kafkit.ProducerConfig
	logger *slog.Logger
}

func newSegmentIOProducer(cfg kafkit.ProducerConfig) kafkit.Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Brokers...),
		Topic:    cfg.Topic,
		Balancer: chooseSegmentIOBalancer(cfg.Balancer),
		Async:    cfg.Async,
	}

	return &segmentIOProducer{
		writer: writer,
		cfg:    cfg,
		logger: cfg.Logger,
	}
}

func chooseSegmentIOBalancer(b kafkit.BalancerStrategy) kafka.Balancer {
	switch b {
	case kafkit.BalancerHash:
		return &kafka.Hash{}
	case kafkit.BalancerMurmur2:
		return &kafka.Murmur2Balancer{}
	case kafkit.BalancerLeastBytes:
		return &kafka.LeastBytes{}
	case kafkit.BalancerManual:
		return &kafka.CRC32Balancer{} //TODO: verify if this is the correct one for manual
	default:
		return &kafka.RoundRobin{}
	}
}

func (s *segmentIOProducer) SendWith(ctx context.Context, value []byte, opts ...kafkit.ProduceOption) error {
	msg := kafkit.ProduceMessage{Value: value}
	for _, o := range opts {
		o(&msg)
	}

	kHeaders := make([]kafka.Header, len(msg.Headers))
	for i, h := range msg.Headers {
		kHeaders[i] = kafka.Header{Key: h.Key, Value: h.Value}
	}

	kmsg := kafka.Message{
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: kHeaders,
	}

	return s.writer.WriteMessages(ctx, kmsg)
}

func (s *segmentIOProducer) Close(ctx context.Context) error {
	if s.writer != nil {
		return s.writer.Close()
	}
	return nil
}
