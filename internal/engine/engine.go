package engine

import (
	"context"
	"fmt"
	"log"

	"github.com/Constant2502/constant-tiny-claw/internal/provider"
	"github.com/Constant2502/constant-tiny-claw/internal/schema"
	"github.com/Constant2502/constant-tiny-claw/internal/tools"
)

type AgentEngine struct {
	provider provider.LLMProvider
	registry tools.Registry

	WorkDir string
}

func NewAgentEngine(provider provider.LLMProvider, registry tools.Registry, workDir string) *AgentEngine {
	return &AgentEngine{
		provider: provider,
		registry: registry,
		WorkDir:  workDir,
	}
}

func (e *AgentEngine) Run(ctx context.Context, userPrompt string) error {
	log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)

	contextHistory := []schema.Message{
		{
			Role:    schema.RoleSystem,
			Content: "You are constant-tiny-claw, an expert coding assistant. You have full access to tools in the workspace.",
		},
		{
			Role:    schema.RoleUser,
			Content: userPrompt,
		},
	}

	turnCount := 0

	for {
		turnCount++
		log.Printf("========== [Turn %d] 开始 ===========\n", turnCount)

		//获取当前挂载的所有工具定义
		availableTools := e.registry.GetAvailableTools()

		//向大模型发起推理请求
		log.Printf("[Engine] 正在思考 (Reasoning)...")
		responseMsg, err := e.provider.Generate(ctx, contextHistory, availableTools)
		if err != nil {
			return fmt.Errorf("模型调用失败: %w", err)
		}

		contextHistory = append(contextHistory, *responseMsg)

		if responseMsg.Content != "" {
			log.Printf("模型: %s", responseMsg.Content)
		}

		//退出条件判断：没有请求任何工具
		if len(responseMsg.ToolCalls) == 0 {
			log.Printf("[Engine] 任务完成，退出循环")
			break
		}

		log.Printf("[Engine] 模型请求调用 %d 个工具...\n", len(responseMsg.ToolCalls))

		for _, toolCall := range responseMsg.ToolCalls {
			log.Printf(" -> 🛠执行工具： %s, 参数: %s\n", toolCall.Name, string(toolCall.Arguments))

			result := e.registry.Execute(ctx, toolCall)

			if result.IsError {
				log.Printf(" -> ❌ 工具执行报错: %s\n", result.Output)
			} else {
				log.Printf(" -> ✅ 工具执行成功 (返回 %d 字节)\n", len(result.Output))
			}

			observationMsg := schema.Message{
				Role:       schema.RoleUser,
				Content:    result.Output,
				ToolCallID: toolCall.ID,
			}
			contextHistory = append(contextHistory, observationMsg)
		}
	}

	return nil
}
