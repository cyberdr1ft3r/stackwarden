package pkg

// OperationResult represents a JSON response from agent operations.
type OperationResult struct {
	OK     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}
