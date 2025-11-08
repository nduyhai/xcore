package franzgo

import "github.com/nduyhai/xcore/pubsub/kafkit"

func init() {
	kafkit.Register(newFranzGo())
}
