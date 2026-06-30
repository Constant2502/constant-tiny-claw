package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Constant2502/constant-tiny-claw/internal/engine"
	"github.com/Constant2502/constant-tiny-claw/internal/schema"
	"github.com/joho/godotenv"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type FeishuBot struct {
	client    *lark.Client
	appID     string
	appSecret string
	engine    *engine.AgentEngine
}

func NewFeishuBot(eng *engine.AgentEngine) *FeishuBot {
	_ = godotenv.Load()
	appID := os.Getenv("FEISHU_APP_ID")
	appSecret := os.Getenv("FEISHU_APP_SECRET")

	if appID == "" || appSecret == "" {
		log.Fatalf("飞书应用 ID 或应用密钥为空")
	}

	client := lark.NewClient(appID, appSecret)

	return &FeishuBot{
		client:    client,
		appID:     appID,
		appSecret: appSecret,
		engine:    eng,
	}
}

func (b *FeishuBot) GetEventDispather() *dispatcher.EventDispatcher {
	_ = godotenv.Load()
	encryptKey := os.Getenv("FEISHU_ENCRYPT_KEY")
	verifyKey := os.Getenv("FEISHU_VERIFY_TOKEN")

	handler := dispatcher.NewEventDispatcher(verifyKey, encryptKey).
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			contentStr := *event.Event.Message.Content
			contentStr = strings.TrimPrefix(contentStr, `{"text":"`)
			contentStr = strings.TrimSuffix(contentStr, `"}"`)

			chatId := *event.Event.Message.ChatId
			log.Printf("[Feishu] 收到会话 %s 消息: %s\n", chatId, contentStr)

			go b.handleAgentRun(chatId, contentStr)

			return nil
		}).
		OnP2MessageReadV1(func(ctx context.Context, event *larkim.P2MessageReadV1) error {
			return nil
		})
	return handler
}

func (b *FeishuBot) handleAgentRun(chatId string, prompt string) {
	//为当前聊天窗口实例化一个专属的Reporter
	reporter := &FeishuReporter{
		client: b.client,
		chatId: chatId,
	}

	workDir, err := os.Getwd()
	if err != nil {
		reporter.sendMsg(fmt.Sprintf("获取当前工作目录失败: %s", err.Error()))
		return
	}

	session := engine.GlobalSessionMgr.GetOrCreate(chatId, workDir)
	session.Append(schema.Message{Role: schema.RoleUser, Content: prompt})

	//启动引擎
	err = b.engine.Run(context.Background(), session, reporter)
	if err != nil {
		reporter.sendMsg(fmt.Sprintf("智能体运行崩溃: %s", err.Error()))
	}
}

type FeishuReporter struct {
	client *lark.Client
	chatId string
}

// SendMessage封装了调用飞书OpenAPI发送卡片或文本的能力
func (r *FeishuReporter) sendMsg(text string) {
	// 构建文本信息内容
	textContent := map[string]string{
		"text": text,
	}
	contentBytes, _ := json.Marshal(textContent)
	contentStr := string(contentBytes)

	msgReq := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.CreateMessageV1ReceiveIDTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(r.chatId).
			MsgType(larkim.MsgTypeText).
			Content(contentStr).
			Build()).
		Build()

	resp, err := r.client.Im.Message.Create(context.Background(), msgReq)
	if err != nil {
		log.Printf("[Feishu] ❌ 发送消息失败: %v\n", err)
	} else {
		msgID := ""
		if resp != nil && resp.Data != nil && resp.Data.MessageId != nil {
			msgID = *resp.Data.MessageId
		}
		log.Printf("[Feishu] ✅ 消息已发送, msgId=%s\n", msgID)
	}
}

func (r *FeishuReporter) OnThinking(ctx context.Context) {
	r.sendMsg("🤔模型正在慢思考(Thinking)...")
}

func (r *FeishuReporter) OnToolCall(ctx context.Context, toolName string, args string) {
	r.sendMsg(fmt.Sprintf("🛠**正在执行工具**: `%s`\n参数:`%s`", toolName, args))
}

func (r *FeishuReporter) OnMessage(ctx context.Context, content string) {
	r.sendMsg(content)
}

func (r *FeishuReporter) OnToolResult(ctx context.Context, toolName string, result string, isError bool) {
	if isError {
		r.sendMsg(fmt.Sprintf("⚠️ **执行报错** (%s)：\n%s", toolName, result))
	} else {
		// 成功时仅汇报成功，不刷全量日志
		r.sendMsg(fmt.Sprintf("✅ **执行成功** (%s)", toolName))
	}
}

var _ engine.Reporter = (*FeishuReporter)(nil)
