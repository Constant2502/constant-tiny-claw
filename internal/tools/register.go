package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Constant2502/constant-tiny-claw/internal/schema"
)

type BaseTools interface {
	// Name 返回工具的全局唯一名称
	Name() string

	// Definition 返回用于提交给大模型的工具元信息和参数 JSON schema
	Definition() schema.ToolDefinition

	// Execute 接收大模型吐出的JSON参数，执行具体业务逻辑
	// 注意：参数是json.RawMessage, 反序列化由各个具体工具内部自行处理
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

type Registry interface {
	// Register 挂载一个新的工具到系统中
	Register(tool BaseTools)

	// GetAvailableTools 返回当前系统挂载的所有工具的Schema，供main loop调用。
	GetAvailableTools() []schema.ToolDefinition

	// Execute 实际路由并执行模型请求的工具调用。
	Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult
}

// Registry接口的默认实现
type registryImpl struct {
	tools map[string]BaseTools
}

func NewRegistry() Registry {
	return &registryImpl{
		tools: make(map[string]BaseTools),
	}
}

func (r *registryImpl) Register(tool BaseTools) {
	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		log.Printf("[Warning] Tool %s already registered", name)
	}
	r.tools[name] = tool
	log.Printf("[Info] Tool %s registered", name)
}

func (r *registryImpl) GetAvailableTools() []schema.ToolDefinition {
	var availableTools []schema.ToolDefinition
	for _, tool := range r.tools {
		availableTools = append(availableTools, tool.Definition())
	}
	return availableTools
}

func (r *registryImpl) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	//1.路由查找。如果在注册表中找不到工具，那就是模型出现了幻觉，直接向模型抛出错误。
	tool, exists := r.tools[call.Name]
	if !exists {
		errMsg := fmt.Sprintf("Tool %s not found", call.Name)
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     errMsg,
			IsError:    true,
		}
	}

	//2. 执行工具逻辑，将原始的JSON字节流直接丢给具体工具。
	output, err := tool.Execute(ctx, call.Arguments)

	//3. 封装结果，将执行结果或底层物理错误封装后返回给Main Loop
	if err != nil {
		errMsg := fmt.Sprintf("Tool %s execute failed with %s", call.Name, err)
		return schema.ToolResult{
			ToolCallID: call.ID,
			Output:     errMsg,
			IsError:    true,
		}
	}

	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     output,
		IsError:    false,
	}
}
