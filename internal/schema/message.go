package schema

import "encoding/json"

type Role string

const (
	// RoleSystem 系统提示词：确定agent性格与红线
	RoleSystem Role = "system"

	// RoleUser 用户输入 工具执行的返回Result
	RoleUser Role = "user"

	// RoleAssistant 模型输出
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	IsError    bool   `json:"is_error"`
}

type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}
