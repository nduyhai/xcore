//go:build integration

package segmentio

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nduyhai/xcore/pubsub/kafkit"
	segkafka "github.com/segmentio/kafka-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

// sanitizeBootstrap converts values like "PLAINTEXT://host:port" to "host:port" which
// is what  segmentio clients expect.
func sanitizeBootstrap(s string) string {
	if i := strings.Index(s, "://"); i != -1 {
		return s[i+3:]
	}
	return s
}

func requireDocker(t *testing.T) {
	// If running with Colima, Testcontainers' reaper (Ryuk) may try to mount the client socket path
	// (e.g. $HOME/.colima/default/docker.sock) into the container, which fails because that path does
	// not exist inside the Linux VM. In that case, instruct Testcontainers to mount the VM socket path
	// instead.
	if dh := os.Getenv("DOCKER_HOST"); strings.Contains(dh, "/.colima/") {
		if os.Getenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE") == "" {
			_ = os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")
		}
	}
	if os.Getenv("TESTCONTAINERS_RYUK_DISABLED") == "1" {
		// still allowed, just info marker
	}
}

func runKafkaContainer(t *testing.T) (context.Context, *tckafka.KafkaContainer, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	t.Logf("[it] starting Kafka container...")
	kcont, err := tckafka.RunContainer(ctx)
	if err != nil {
		t.Fatalf("failed to start kafka container: %v", err)
	}
	t.Logf("[it] Kafka container started")
	t.Cleanup(func() {
		_ = kcont.Terminate(context.Background())
	})

	// Prefer Brokers(ctx) if available; fall back to BootstrapServers-style if needed
	var bootstrap string
	if brokers, berr := kcont.Brokers(ctx); berr == nil && len(brokers) > 0 {
		bootstrap = brokers[0]
		t.Logf("[it] Kafka broker: %s", bootstrap)
	} else {
		// Older/newer APIs may differ; try Address or BootstrapServers if present via reflection-like safe calls
		// Since we can't reflect here easily, surface the original error for visibility
		if berr != nil {
			t.Fatalf("failed to get brokers: %v", berr)
		}
	}
	return ctx, kcont, sanitizeBootstrap(bootstrap)
}

// ensureTopic creates a topic and waits until it appears in metadata with a leader.
func ensureTopic(ctx context.Context, bootstrap, topic string, partitions, replication int) error {
	// Connect to any broker via the given bootstrap address.
	conn, err := segkafka.DialContext(ctx, "tcp", bootstrap)
	if err != nil {
		return err
	}
	defer func(conn *segkafka.Conn) {
		_ = conn.Close()
	}(conn)

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	cAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	cconn, err := segkafka.DialContext(ctx, "tcp", cAddr)
	if err != nil {
		return err
	}
	defer func(cconn *segkafka.Conn) {
		_ = cconn.Close()
	}(cconn)

	if err := cconn.CreateTopics(segkafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: replication,
	}); err != nil {
		return err
	}

	// Wait until topic shows up with partitions and leaders
	deadline := time.Now().Add(10 * time.Second)
	for {
		parts, err := conn.ReadPartitions(topic)
		if err == nil && len(parts) >= partitions {
			// Ensure leaders
			ready := true
			for _, p := range parts {
				if p.Leader.Host == "" || p.Leader.Port == 0 {
					ready = false
					break
				}
			}
			if ready {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("topic %s not ready in time", topic)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func TestIntegration_SegmentIO_ProduceConsume(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	requireDocker(t)

	ctx, _, bootstrap := runKafkaContainer(t)

	topic := "it-seg-" + time.Now().Format("150405.000000")
	group := "it-seg-group-" + time.Now().Format("150405.000000")

	// Explicitly create a topic in case the broker disables auto-creation
	if err := ensureTopic(ctx, bootstrap, topic, 4, 1); err != nil {
		t.Fatalf("ensureTopic: %v", err)
	}

	msgCh := make(chan kafkit.ConsumeMessage, 1)
	h := kafkit.HandlerFunc(func(ctx context.Context, msg kafkit.ConsumeMessage) error {
		select {
		case msgCh <- msg:
		default:
		}
		return nil
	})

	factory, err := kafkit.GetFactory(BackendSegmentIO)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	consumer, err := factory.NewConsumer([]string{bootstrap}, topic, group, h,
		kafkit.WithCommitInterval(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	prod, err := factory.NewProducer([]string{bootstrap}, topic)
	if err != nil {
		t.Fatalf("producer: %v", err)
	}

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _ = consumer.Start(cctx) }()
	defer consumer.Close(context.Background())
	defer prod.Close(context.Background())

	// allow join
	time.Sleep(2 * time.Second)

	payload := []byte("hello-segmentio")
	if err := prod.SendWith(ctx, payload, kafkit.WithKey([]byte("k1"))); err != nil {
		t.Fatalf("produce: %v", err)
	}

	select {
	case got := <-msgCh:
		if string(got.Value) != string(payload) {
			t.Fatalf("unexpected value: %q", string(got.Value))
		}
		if got.Topic != topic {
			t.Fatalf("unexpected topic: %s", got.Topic)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("did not receive message in time")
	}
}
func TestIntegration_SegmentIO_HashBalancer_SameKeySamePartition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	requireDocker(t)

	ctx, _, bootstrap := runKafkaContainer(t)

	topic := "it-seg-hash-" + time.Now().Format("150405.000000")
	group := "it-seg-hash-group-" + time.Now().Format("150405.000000")

	if err := ensureTopic(ctx, bootstrap, topic, 4, 1); err != nil {
		t.Fatalf("ensureTopic: %v", err)
	}

	partsCh := make(chan int, 2)
	targetKey := []byte("same-key")
	h := kafkit.HandlerFunc(func(ctx context.Context, msg kafkit.ConsumeMessage) error {
		if string(msg.Key) == string(targetKey) {
			select {
			case partsCh <- msg.Partition:
			default:
			}
		}
		return nil
	})

	factory, err := kafkit.GetFactory(BackendSegmentIO)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	consumer, err := factory.NewConsumer([]string{bootstrap}, topic, group, h,
		kafkit.WithCommitInterval(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	prod, err := factory.NewProducer([]string{bootstrap}, topic, kafkit.WithBalancer(kafkit.BalancerHash))
	if err != nil {
		t.Fatalf("producer: %v", err)
	}

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _ = consumer.Start(cctx) }()
	defer consumer.Close(context.Background())
	defer prod.Close(context.Background())

	// allow join
	time.Sleep(3 * time.Second)

	if err := prod.SendWith(ctx, []byte("m1"), kafkit.WithKey(targetKey)); err != nil {
		t.Fatalf("produce m1: %v", err)
	}
	if err := prod.SendWith(ctx, []byte("m2"), kafkit.WithKey(targetKey)); err != nil {
		t.Fatalf("produce m2: %v", err)
	}

	var p1, p2 int
	select {
	case p1 = <-partsCh:
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting first message")
	}
	select {
	case p2 = <-partsCh:
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting second message")
	}
	if p1 != p2 {
		t.Fatalf("expected same partition for same key, got %d and %d", p1, p2)
	}
}
