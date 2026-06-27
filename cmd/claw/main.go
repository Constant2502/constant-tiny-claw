package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Constant2502/constant-tiny-claw/internal/engine"
	"github.com/Constant2502/constant-tiny-claw/internal/feishu"
	"github.com/Constant2502/constant-tiny-claw/internal/provider"
	"github.com/Constant2502/constant-tiny-claw/internal/schema"
	"github.com/Constant2502/constant-tiny-claw/internal/tools"
	"github.com/joho/godotenv"
	"github.com/larksuite/oapi-sdk-go/v3/core/httpserverext"
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
	_ = godotenv.Load()
	if os.Getenv("ZHIPU_API_KEY") == "" {
		log.Fatal("请先导入智谱API的环境变量")
	}

	workDir, _ := os.Getwd()

	llmProvider := provider.NewZhipuOpenAIProvider("glm-4.5-air")

	registry := tools.NewRegistry()

	readFileTool := tools.NewReadFileTool(workDir)
	writeFileTool := tools.NewWriteFileTool(workDir)
	bashTool := tools.NewBashTool(workDir)
	editFileTool := tools.NewEditFileTool(workDir)

	registry.Register(readFileTool)
	registry.Register(writeFileTool)
	registry.Register(bashTool)
	registry.Register(editFileTool)

	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	bot := feishu.NewFeishuBot(eng)
	handler := httpserverext.NewEventHandlerFunc(bot.GetEventDispather())

	http.HandleFunc("/webhook/event", handler)

	port := ":48000"
	log.Printf("🚀 go-tiny-claw 飞书服务端已启动，正在监听 %s 端口\n", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
