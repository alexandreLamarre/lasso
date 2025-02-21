package tracing

var (
	distributedTracingEnabled = false
)

func SetDistributedTracingEnabled(enabled bool) {
	distributedTracingEnabled = enabled
}

func IsDistributedTracingEnabled() bool {
	return distributedTracingEnabled
}
