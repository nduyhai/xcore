module github.com/nduyhai/xcore/error/gerror

go 1.24.5

require (
	github.com/nduyhai/xcore/error/xerr v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.75.1
	google.golang.org/protobuf v1.36.9
)

require (
	golang.org/x/sys v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250908214217-97024824d090 // indirect
)

replace github.com/nduyhai/xcore/error/xerr => ../xerr
