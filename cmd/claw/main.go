package main

import (
	"context"
	"log"
	"os"

	"github.com/Constant2502/constant-tiny-claw/internal/engine"
	"github.com/Constant2502/constant-tiny-claw/internal/provider"
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

func (m *mockRegistry) GetAvailableTools() []schema.ToolDefinition {
	return []schema.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "获取当前指定城市天气状况",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"city"},
			},
		},
	}
}

func (m *mockRegistry) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	log.Printf(" -> [Mock工具执行]获取 %s 的天气中....\n", call.Name)
	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     "API返回：今天是晴天气温二十五度。",
		IsError:    false,
	}
}

func main() {
	if os.Getenv("ZHIPU_API_KEY") == "" {
		log.Fatal("请先导入智谱API的环境变量")
	}

	workDir, _ := os.Getwd()

	llmProvider := provider.NewZhipuOpenAIProvider("glm-4.5-air")

	registry := &mockRegistry{}

	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	prompt := "想去深圳跑步，帮我查查天气合适吗？"

	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatal(err)
	}
}
