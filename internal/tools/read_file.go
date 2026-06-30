package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Constant2502/constant-tiny-claw/internal/schema"
)

type ReadFileTool struct {
	//将引擎的工作目录注入给工具，限制它只能在此目录及其子目录下操作。
	workDir string
}

func NewReadFileTool(workDir string) *ReadFileTool {
	return &ReadFileTool{workDir: workDir}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        t.Name(),
		Description: "读取指定路径的文件内容，请提供相对工作区的路径。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "要读取的文件路径,如 cmd/claw/main.go",
				},
			},
			"required": []string{"path"},
		},
	}
}

type readFileArgs struct {
	Path string `json:"path"`
}

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	//1. 延迟解析，将大模型传过来的JSON参数解析为强类型结构体。
	var input readFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("param parse error: %w", err)
	}

	//2. 拼接绝对路径。
	fullPath := filepath.Join(t.workDir, input.Path)

	//3. 执行物理I/O操作。
	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("open file error: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read file error: %w", err)
	}

	//4. 核心防线长度截断保护。
	//为了防止大模型读取几百兆的日志文件，导致Context瞬间爆炸，我们在工具内部直接进行物理截断。
	const maxLen = 8000
	if len(content) > maxLen {
		truncateMsg := fmt.Sprintf("%s\n\n...[由于内容过长，已被系统截断至前%d字节。]", string(content[:maxLen]), maxLen)
		return truncateMsg, nil
	}
	return string(content), nil
}
