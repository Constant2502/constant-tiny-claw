package engine

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Constant2502/constant-tiny-claw/internal/provider"
	"github.com/Constant2502/constant-tiny-claw/internal/schema"
	"github.com/Constant2502/constant-tiny-claw/internal/tools"
)

type AgentEngine struct {
	provider       provider.LLMProvider
	registry       tools.Registry
	WorkDir        string
	EnableThinking bool
}

func NewAgentEngine(provider provider.LLMProvider, registry tools.Registry, workDir string, enableThinking bool) *AgentEngine {
	return &AgentEngine{
		provider:       provider,
		registry:       registry,
		WorkDir:        workDir,
		EnableThinking: enableThinking,
	}
}

func (e *AgentEngine) Run(ctx context.Context, userPrompt string) error {
	log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)
	log.Printf("[Engine] 慢思考模式: %v\n", e.EnableThinking)

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
		if e.EnableThinking {
			log.Printf("[Engine][Phase 1] 剥夺工具使用权，进入慢思考与规划阶段")

			thinkResp, err := e.provider.Generate(ctx, contextHistory, nil)
			if err != nil {
				return fmt.Errorf("Thinking failed: %v", err)
			}

			if thinkResp.Content != "" {
				log.Printf("[内部思考 Trace] %s", thinkResp.Content)
				contextHistory = append(contextHistory, *thinkResp)
			}

		}
		log.Printf("[Engine][Phase 2] 恢复工具挂载，等待模型行动...")
		actionResp, err := e.provider.Generate(ctx, contextHistory, availableTools)
		if err != nil {
			return fmt.Errorf("Action阶段生成失败: %w", err)
		}

		contextHistory = append(contextHistory, *actionResp)

		if actionResp.Content != "" {
			log.Printf("[对外回复]: %s", actionResp.Content)
		}

		//退出条件判断：没有请求任何工具
		if len(actionResp.ToolCalls) == 0 {
			log.Printf("[Engine] 任务完成，退出循环")
			break
		}

		log.Printf("[Engine] 模型并发请求调用 %d 个工具...\n", len(actionResp.ToolCalls))

		//核心改造，从串行改成并行。
		//1.预分配一个固定长度的切片，用于安全地存放各个并发工具的执行结果，长度和tool calls的数量完全一致。
		observationMsgs := make([]schema.Message, len(actionResp.ToolCalls))

		//2.声明WaitGroup用于阻塞等待所有协程完成。
		var wg sync.WaitGroup

		for i, toolCall := range actionResp.ToolCalls {
			wg.Add(1)

			go func(idx int, call schema.ToolCall) {
				defer wg.Done()

				log.Printf("    -> [Go-%d] 🛠 触发并发执行:  %s\n", idx, call.Name)

				result := e.registry.Execute(ctx, call)

				if result.IsError {
					log.Printf("    -> [Go-%d] ❌ 工具执行报错:  %s\n", idx, call.Name)
				} else {
					log.Printf("    -> [Go-%d] ✅ 工具执行成功（返回 %d 字节)  \n", idx, len(result.Output))
				}

				obsMsg := schema.Message{
					Role:       schema.RoleUser,
					Content:    result.Output,
					ToolCallID: call.ID,
				}

				observationMsgs[idx] = obsMsg
			}(i, toolCall)
		}

		wg.Wait()
		log.Println("[Engine] 所有并发工具执行完毕，开始聚合观察结果(Observation)...")

		for _, obs := range observationMsgs {
			contextHistory = append(contextHistory, obs)
		}
		//for _, toolCall := range actionResp.ToolCalls {
		//	log.Printf(" -> 🛠执行工具： %s, 参数: %s\n", toolCall.Name, string(toolCall.Arguments))
		//
		//	result := e.registry.Execute(ctx, toolCall)
		//
		//	if result.IsError {
		//		log.Printf(" -> ❌ 工具执行报错: %s\n", result.Output)
		//	} else {
		//		log.Printf(" -> ✅ 工具执行成功 (返回 %d 字节)\n", len(result.Output))
		//	}
		//
		//	observationMsg := schema.Message{
		//		Role:       schema.RoleUser,
		//		Content:    result.Output,
		//		ToolCallID: toolCall.ID,
		//	}
		//	contextHistory = append(contextHistory, observationMsg)
		//}
	}

	return nil
}
