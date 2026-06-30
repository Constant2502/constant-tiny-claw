package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Constant2502/constant-tiny-claw/internal/schema"
)

type EditFileTool struct {
	workDir string
}

func (e *EditFileTool) Name() string {
	return "edit"
}

func NewEditFileTool(workDir string) *EditFileTool {
	return &EditFileTool{
		workDir: workDir,
	}
}

func (e *EditFileTool) Definition() schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        e.Name(),
		Description: "对现有文件进行局部的字符串替换，这比重写整个文件更安全、更快速。请提供足够的old text上下文，以确保匹配的唯一性。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "要修改的文件路径。",
				},
				"old_text": map[string]interface{}{
					"type":        "string",
					"description": "文件中原有的文本必须包含足够的上下文(建议上下各多包含几行)，以确保在文件中的唯一性。",
				},
				"new_text": map[string]interface{}{
					"type":        "string",
					"description": "要替换成的新文本。",
				},
			},
			"required": []string{"path", "old_text", "new_text"},
		},
	}
}

type editFileArgs struct {
	Path    string `json:"path"`
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}

func fuzzyReplace(originalContext, oldText, newText string) (string, error) {
	// L1: 精确匹配
	count := strings.Count(originalContext, oldText)
	if count == 1 {
		return strings.Replace(originalContext, oldText, newText, 1), nil
	}
	if count > 1 {
		return "", fmt.Errorf("old_text匹配到了%d处，请提供更多的上下文代码", count)
	}

	// L2:换行符规一化(统一将\r\n转换为\n）
	normalizedContent := strings.ReplaceAll(originalContext, "\r\n", "\n")
	normalizedOld := strings.ReplaceAll(oldText, "\r\n", "\n")

	count = strings.Count(normalizedContent, normalizedOld)
	if count == 1 {
		return strings.Replace(normalizedContent, normalizedOld, newText, 1), nil
	}

	// L3: Trim Space匹配（忽略首尾的空行和空格）
	trimmedOld := strings.TrimSpace(normalizedOld)
	if trimmedOld != "" {
		count = strings.Count(normalizedContent, trimmedOld)
		if count == 1 {
			return strings.Replace(normalizedContent, trimmedOld, newText, 1), nil
		}
	}

	//L4: 逐行去缩进匹配(最强力的容错:消除大模型遗漏缩进的幻觉。)
	return lineByLineReplace(normalizedContent, normalizedOld, newText)
}

// lineByLineReplace 将文本按行切割，去除首尾空白后进行滑动窗口匹配
func lineByLineReplace(content, oldText, newText string) (string, error) {
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(strings.TrimSpace(oldText), "\n")

	if len(oldLines) == 0 || len(contentLines) < len(oldLines) {
		return "", fmt.Errorf("找不到该代码片段")
	}

	// 清理 oldLines 的每行首尾空白
	for i := range oldLines {
		oldLines[i] = strings.TrimSpace(oldLines[i])
	}

	matchCount := 0
	matchStartIndex := -1
	matchEndIndex := -1

	// 滑动窗口在原始文件中寻找匹配块
	for i := 0; i <= len(contentLines)-len(oldLines); i++ {
		isMatch := true
		for j := 0; j < len(oldLines); j++ {
			if strings.TrimSpace(contentLines[i+j]) != oldLines[j] {
				isMatch = false
				break
			}
		}

		if isMatch {
			matchCount++
			matchStartIndex = i
			matchEndIndex = i + len(oldLines)
		}
	}

	if matchCount == 0 {
		return "", fmt.Errorf("在文件中未找到 old_text，请大模型先调用 read_file 仔细确认文件内容和缩进")
	}
	if matchCount > 1 {
		return "", fmt.Errorf("模糊匹配到了 %d 处相似代码，请提供更多上下行代码以精确定位", matchCount)
	}

	// 执行替换：将匹配到的原始行范围替换为 newText 拆分后的行
	// (这里简单处理，将 newText 直接作为整体替换进去)
	var newContentLines []string
	newContentLines = append(newContentLines, contentLines[:matchStartIndex]...)
	newContentLines = append(newContentLines, newText) // 插入新内容
	newContentLines = append(newContentLines, contentLines[matchEndIndex:]...)

	return strings.Join(newContentLines, "\n"), nil
}

// internal/tools/edit_file.go (续)

func (t *EditFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var input editFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	fullPath := filepath.Join(t.workDir, input.Path)

	// 1. 读取原文件内容
	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败，请确认路径是否正确: %w", err)
	}
	originalContent := string(contentBytes)

	// 2. 调用多级模糊替换算法
	newContent, err := fuzzyReplace(originalContent, input.OldText, input.NewText)
	if err != nil {
		// 【驾驭哲学】将具体的报错原因 (如匹配到多处) 原样返回，让大模型自行纠正
		return "", err
	}

	// 3. 将新内容安全地写回磁盘
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("写回文件失败: %w", err)
	}

	return fmt.Sprintf("✅ 成功修改文件: %s", input.Path), nil
}
