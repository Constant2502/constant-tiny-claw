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

func (e *AgentEngine) Run(ctx context.Context, userPrompt string, reporter Reporter) error {
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
		if e.EnableThinking {
			if reporter != nil {
				reporter.OnThinking(ctx)
			}

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

		if actionResp.Content != "" && reporter != nil {
			reporter.OnMessage(ctx, actionResp.Content)
		}

		//退出条件判断：没有请求任何工具
		if len(actionResp.ToolCalls) == 0 {
			log.Printf("[Engine] 任务完成，退出循环")
			break
		}

		observationMsgs := make([]schema.Message, len(actionResp.ToolCalls))

		//2.声明WaitGroup用于阻塞等待所有协程完成。
		var wg sync.WaitGroup

		for i, toolCall := range actionResp.ToolCalls {
			wg.Add(1)

			go func(idx int, call schema.ToolCall) {
				defer wg.Done()

				if reporter != nil {
					reporter.OnToolCall(ctx, call.Name, string(call.Arguments))
				}

				result := e.registry.Execute(ctx, call)

				if reporter != nil {
					displayOutput := result.Output
					if len(displayOutput) > 200 {
						displayOutput = displayOutput[:200] + "...(已截断)"
					}
					reporter.OnToolResult(ctx, call.Name, displayOutput, result.IsError)
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

		for _, obs := range observationMsgs {
			contextHistory = append(contextHistory, obs)
		}
	}

	return nil
}
