package pluginruntime

// Logical operation names (stable for Go / Deno / Rust SDKs).
const (
	OpHandshake       = "handshake"
	OpHealth        = "health"
	OpReady         = "ready"
	OpInit          = "init"
	OpExecuteSource = "execute_source"
	OpExecuteProcess = "execute_process"
	OpExecuteDispatch = "execute_dispatch"
	OpTestConfig    = "test_config"
)
