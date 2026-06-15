package tools

import (
	"context"

	"github.com/Constant2502/constant-tiny-claw/internal/schema"
)

type Registry interface {
	GetAvailableTools() []schema.ToolDefinition

	Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult
}
