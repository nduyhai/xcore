package segmentio

import "github.com/nduyhai/xcore/pubsub/kafkit"

func init() {
	kafkit.Register(newSegmentIO())
}
