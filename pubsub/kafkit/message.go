package kafkit

import "time"

type ProduceMessage struct {
	Key     []byte
	Value   []byte
	Headers []Header
}

type ConsumeMessage struct {
	Topic     string
	Partition int
	Offset    int64
	Timestamp time.Time
	Key       []byte
	Value     []byte
	Headers   []Header
}

type Header struct {
	Key   string
	Value []byte
}
