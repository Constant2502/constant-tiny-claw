package provider

import (
	"context"

	"github.com/Constant2502/constant-tiny-claw/internal/schema"
)

type LLMProvider interface {
	Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error)
}
