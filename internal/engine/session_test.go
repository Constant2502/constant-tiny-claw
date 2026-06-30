package engine

import (
	"testing"

	"github.com/Constant2502/constant-tiny-claw/internal/schema"
)

func TestGetWorkingMemoryKeepsLatestUserTurn(t *testing.T) {
	session := NewSession("test", ".")
	session.Append(
		schema.Message{Role: schema.RoleUser, Content: "执行三步任务"},
		schema.Message{Role: schema.RoleAssistant, Content: "调用 echo", ToolCalls: []schema.ToolCall{{ID: "call_1", Name: "bash"}}},
		schema.Message{Role: schema.RoleUser, Content: "开始排查日志", ToolCallID: "call_1"},
		schema.Message{Role: schema.RoleAssistant, Content: "读取日志", ToolCalls: []schema.ToolCall{{ID: "call_2", Name: "read_file"}}},
		schema.Message{Role: schema.RoleUser, Content: "很长的日志", ToolCallID: "call_2"},
		schema.Message{Role: schema.RoleAssistant, Content: "调用 date", ToolCalls: []schema.ToolCall{{ID: "call_3", Name: "bash"}}},
		schema.Message{Role: schema.RoleUser, Content: "Tue Jun 30 08:02:03 CST 2026", ToolCallID: "call_3"},
	)

	memory := session.GetWorkingMemory(6)
	if len(memory) != 7 {
		t.Fatalf("期望保留完整用户轮次 7 条消息，实际得到 %d 条", len(memory))
	}
	if memory[0].Role != schema.RoleUser || memory[0].ToolCallID != "" {
		t.Fatalf("期望第一条消息是真实用户请求，实际为 role=%s tool_call_id=%q", memory[0].Role, memory[0].ToolCallID)
	}
}
