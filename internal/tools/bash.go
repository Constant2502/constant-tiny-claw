package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/Constant2502/constant-tiny-claw/internal/schema"
)

type BashTool struct {
	workDir string
}

func NewBashTool(workDir string) *BashTool {
	return &BashTool{workDir: workDir}
}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: "在当前工作区执行任意的Bash命令，支持链式命令，返回标准输出和标准错误。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "要执行的bash命令，例如: ls -la 或 go test ./...",
				},
			},
			"required": []string{"command"},
		},
	}
}

type bashArgs struct {
	Command string `json:"command"`
}

func (t *BashTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var input bashArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("failed to parse args: %w", err)
	}

	//1.最大执行时间限制
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	//macOS/linux 将指令包裹在 bash -c中执行，以支持环境变量、管道和逻辑与(&&)等复杂Shell语法
	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", input.Command)

	cmd.Dir = t.workDir

	out, err := cmd.CombinedOutput()
	outputStr := string(out)

	if timeoutCtx.Err() == context.DeadlineExceeded {
		return outputStr + "\n[Warning: command execute overtime(30s)]", nil
	}

	if err != nil {
		return "", fmt.Errorf("failed to execute bash: %w", err)
	}

	if outputStr == "" {
		return "command execute success", nil
	}

	const maxLen = 8000
	if len(outputStr) > maxLen {
		return fmt.Sprintf("%s\n\n...[终端输出过长，已截断至前%d字节。", outputStr[:maxLen], len(outputStr)), nil
	}

	return outputStr, nil
}
