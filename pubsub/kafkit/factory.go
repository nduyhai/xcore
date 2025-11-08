package kafkit

import "fmt"

// BalancerStrategy defines how messages are distributed across Kafka partitions.
type BalancerStrategy string

const (
	// BalancerRoundRobin evenly distributes messages across all partitions.
	// ⚙️ Good for high-throughput, unordered workloads (e.g. logs, metrics).
	BalancerRoundRobin BalancerStrategy = "round_robin"

	// BalancerHash uses a hash of the message key to pick the same partition for identical keys.
	// ⚙️ Ensures ordering per key (e.g. same userID or transactionID).
	BalancerHash BalancerStrategy = "hash"

	// BalancerSticky keeps sending messages to the same partition until the batch is full,
	// then switches — improves batching efficiency and throughput.
	// ⚙️ Default in modern Kafka Java/Confluent and Franz-Go clients.
	BalancerSticky BalancerStrategy = "sticky"

	// BalancerMurmur2 uses Kafka’s Java-compatible Murmur2 hash algorithm.
	// ⚙️ Guarantees consistent partitioning across languages (Go ↔ Java ↔ Python).
	BalancerMurmur2 BalancerStrategy = "murmur2"

	// BalancerLeastBytes sends messages to the partition with the least buffered data.
	// ⚙️ Balances load dynamically based on partition lag (SegmentIO only).
	BalancerLeastBytes BalancerStrategy = "least_bytes"

	// BalancerManual allows explicit partition control per message.
	// ⚙️ Used for control topics, test cases, or system streams.
	BalancerManual BalancerStrategy = "manual"
)

type Backend string

type Factory interface {
	Name() Backend
	NewProducer(brokers []string, topic string, opts ...ProducerOption) (Producer, error)
	NewConsumer(brokers []string, topic string, groupID string, handler Handler, opts ...ConsumerOption) (Consumer, error)
}

var backends = map[Backend]Factory{}

func Register(f Factory) {
	backends[f.Name()] = f
}

func GetFactory(name Backend) (Factory, error) {
	if f, ok := backends[name]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("kafka backend not found: %s", name)
}
