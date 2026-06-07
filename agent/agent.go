package agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

type PuzzleAgent struct {
	agent *adk.ChatModelAgent
	store *Store[*schema.Message]
}

func NewAgent(ctx context.Context, config *adk.ChatModelAgentConfig, store *Store[*schema.Message]) *PuzzleAgent {
	agent, err := adk.NewChatModelAgent(ctx, config)
	if err != nil {
		fmt.Printf("Error creating chat model agent: %v", err)
		return nil
	}
	return &PuzzleAgent{agent: agent, store: store}
}

func NewDefaultAgent(ctx context.Context, model model.ToolCallingChatModel, db *gorm.DB, store *Store[*schema.Message]) *PuzzleAgent {
	ddgTool, err := NewDefaultDuckDuckGoTool(ctx)
	if err != nil {
		fmt.Printf("Error creating DuckDuckGo tool: %v", err)
		return nil
	}
	puzzleDBTool, err := NewPuzzleDBTool(ctx, db)
	if err != nil {
		fmt.Printf("Error creating PuzzleDBTool: %v", err)
		return nil
	}

	instruction :=
		`你是一个专业的海龟汤智能助手。请遵循以下原则与用户交流：
		1. 【角色定位】作为海龟汤游戏的主持人，负责引导用户进行游戏，使用“是”/“否”/“不重要”回答用户的问题,在遇到部分正确的情况时,请说明"是或不是"并可以建议用户提问相关内容
		2. 【游戏规则】你作为主持人，有时需要提供汤面，用户需要根据汤面进行游戏，注意提供汤面时不要展示汤底
		3. 【回答风格】语言简洁清晰，逻辑严谨，必要时分点陈述；避免过度冗长
		4. 【安全合规】拒绝生成违法、有害、歧视性或侵犯隐私的内容
		5. 【上下文理解】充分利用对话历史，保持回复的连贯性和一致性
		6. 【多轮交互】当用户需求模糊时，主动提问以澄清意图，而非猜测作答

		## 可使用的工具
		- DuckDuckGo 搜索：用于一般信息查询、最新动态、专业知识查询
		- PuzzleDBTool：用于随机查询数据库中的汤面

		现在，请开始与用户对话。`
	return NewAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PuzzleAgent",
		Description: "海龟汤智能助手,作为主持人,负责引导用户进行海龟汤游戏",
		Instruction: instruction,
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{ddgTool, puzzleDBTool},
			},
		},
	}, store)
}

func (pa *PuzzleAgent) OutputMessage(ctx context.Context, id string, input string, streamFunc func(string), options ...adk.AgentRunOption) {
	sess, err := pa.store.GetOrCreate(id)
	if err != nil {
		fmt.Printf("Error getting session: %v", err)
		return
	}
	sess.Append(&schema.Message{Role: schema.User, Content: input})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: pa.agent, EnableStreaming: true})
	var messageText strings.Builder
	for _, msg := range sess.GetMessages() {
		messageText.WriteString(msg.Content + "\n")
	}
	iter := runner.Query(ctx, messageText.String(), options...)
	pa.stream(id, iter, streamFunc)
}

func (pa *PuzzleAgent) stream(id string, iter *adk.AsyncIterator[*adk.AgentEvent], streamFunc func(string)) {
	sess, err := pa.store.GetOrCreate(id)
	if err != nil {
		fmt.Printf("Error getting session: %v", err)
		return
	}
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			log.Printf("Runner 错误: %v", event.Err)
			break
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			mo := event.Output.MessageOutput

			// 【关键修复 1】：检查 MessageStream 是否为 nil
			if mo.MessageStream == nil {
				// 如果是非流式消息（例如某些特殊事件，或模型直接返回了非流式结果），处理 mo.Message
				if mo.Message != nil && mo.Message.Content != "" {
					//streamFunc(mo.Message.Content)
					sess.Append(mo.Message)
				}
				continue // 跳过 nil stream
			}

			// 只有 Stream 不为 nil 时，才进入流式处理
			pa.streamMessage(sess, mo.MessageStream, streamFunc)
		}
	}
}

func (pa *PuzzleAgent) streamMessage(sess *Session[*schema.Message], s adk.MessageStream, streamFunc func(string)) {
	// 【关键修复 2】：防御性判空，防止外部意外传入 nil 导致 panic
	if s == nil {
		return
	}
	defer s.Close()

	var sb strings.Builder

	for {
		chunk, err := s.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Stream Recv 错误: %v", err)
			break
		}

		// 处理思考过程 (如果开启了 DeepSeek-R1 等支持 Reasoning 的模型)
		if chunk.ReasoningContent != "" {
			// streamFunc(chunk.ReasoningContent)
			// sess.Append(&schema.Message{Role: schema.Assistant, Content: chunk.ReasoningContent})
			// continue
		}

		// 过滤空内容和纯工具调用（工具调用由 Runner 自动处理，不需要输出给前端）
		if chunk.Content == "" && len(chunk.ToolCalls) == 0 {
			continue
		}

		// 只输出文本内容给前端
		if chunk.Content != "" {
			streamFunc(chunk.Content)
			sb.WriteString(chunk.Content)
		}
	}

	// 只有当有实际文本内容时，才追加到历史记录中
	if sb.Len() > 0 {
		sess.Append(&schema.Message{Role: schema.Assistant, Content: sb.String()})
	}
}

func (pa *PuzzleAgent) DelSess(ctx context.Context, id string) error {
	return pa.store.Delete(id)
}
