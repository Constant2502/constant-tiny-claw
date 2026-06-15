package main

import (
	"context"
	"log"
	"os"

	"github.com/Constant2502/constant-tiny-claw/internal/engine"
	"github.com/Constant2502/constant-tiny-claw/internal/schema"
)

type mockProvider struct {
	turn int
}

func (m *mockProvider) Generate(ctx context.Context, msgs []schema.Message, _ []schema.ToolDefinition) (*schema.Message, error) {
	m.turn++
	if m.turn == 1 {
		return &schema.Message{
			Role:    schema.RoleAssistant,
			Content: "看看目录下有什么文件",
			ToolCalls: []schema.ToolCall{
				{ID: "call_123", Name: "bash", Arguments: []byte(`{"command": "ls -la"`)},
			},
		}, nil
	}

	return &schema.Message{
		Role:    schema.RoleAssistant,
		Content: "我看到了文件列表，里面有main.go, 任务完成！",
	}, nil
}

type mockRegistry struct{}

func (m *mockRegistry) GetAvailableTools() []schema.ToolDefinition { return nil }

func (m *mockRegistry) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     "-rw-r--r-- 1 user group 234 Oct 24 10:00 main.go\n",
		IsError:    false,
	}
}

func main() {
	workDir, _ := os.Getwd()

	p := &mockProvider{}
	r := &mockRegistry{}

	eng := engine.NewAgentEngine(p, r, workDir)

	err := eng.Run(context.Background(), "检查当前目录")
	if err != nil {
		log.Fatalf("engine crashed: %v", err)
	}
}
