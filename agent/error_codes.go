package agent

const (
	// AgentError codes
	ErrCodeCheckpointLoad = "checkpoint_load_error"
	ErrCodeLLMInvoke      = "llm_invoke_error"
	ErrCodeMaxIterations  = "max_iterations_reached"

	// ToolResult codes
	ToolCodeHITLRejected   = "hitl_rejected"
	ToolCodeExecutionError = "tool_execution_error"
	ToolCodeSerialization  = "serialization_error"
)
