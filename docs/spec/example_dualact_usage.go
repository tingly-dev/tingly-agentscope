package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/model/openai"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// 示例：使用 Dual Act Agent 构建一个代码生成和测试的工作流

func main() {
	// 1. 创建模型客户端
	modelClient := openai.NewClient(&model.ChatModelConfig{
		ModelName: "gpt-4o",
		APIKey:    "your-api-key",
	})

	// 2. 创建工具集（给 R 使用）
	toolkit := tool.NewToolkit()

	// 注册示例工具：文件写入、代码执行等
	toolkit.Register(&FileWriteTool{}, &tool.RegisterOptions{
		GroupName: "filesystem",
	})
	toolkit.Register(&CodeExecuteTool{}, &tool.RegisterOptions{
		GroupName: "execution",
	})

	// 3. 创建 H (Human-like 决策代理)
	// H 的角色：评估 R 的工作结果，决定下一步
	humanAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name: "planner",
		SystemPrompt: `你是规划代理。你的职责是：
1. 审查执行代理完成的工作
2. 判断任务是否完成
3. 如果未完成，给出清晰的下一步指令

决策规则：
- 如果代码已生成且测试通过 → TERMINATE（终止）
- 如果代码生成但测试失败 → CONTINUE（继续修复）
- 如果方向错误 → REDIRECT（重定向，说明新方案）`,
		Model:         modelClient,
		Memory:        memory.NewHistory(50),
		MaxIterations: 3, // H 的思考不需要太多迭代
	})

	// 4. 创建 R (Reactive 执行代理)
	// R 的角色：执行具体任务，调用工具
	reactiveAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name: "executor",
		SystemPrompt: `你是执行代理。你的职责是：
1. 理解规划代理给你的任务
2. 使用可用工具完成任务
3. 完成后提供清晰的工作总结

你拥有以下能力：
- 写入文件
- 执行代码
- 运行测试`,
		Model:         modelClient,
		Toolkit:       toolkit,
		Memory:        memory.NewHistory(100),
		MaxIterations: 10, // R 可以进行多步工具调用
	})

	// 5. 创建 Dual Act Agent
	dualAct := agent.NewDualActAgent(&agent.DualActConfig{
		Human:      humanAgent,
		Reactive:   reactiveAgent,
		MaxHRLoops: 5, // 最多 5 轮 H-R 交互
	})

	ctx := context.Background()

	// 6. 用户输入任务
	userMsg := message.NewMsg(
		"user",
		"创建一个 Python 函数计算斐波那契数列，并编写测试验证",
		types.RoleUser,
	)

	// 7. 执行 - 框架自动协调 H 和 R 的交互
	fmt.Println("=== 开始 Dual Act 执行 ===\n")

	response, err := dualAct.Reply(ctx, userMsg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n=== 最终结果 ===")
	fmt.Println(response.GetTextContent())
}

// ============ 示例工具实现 ============

// FileWriteTool 文件写入工具
type FileWriteTool struct{}

func (f *FileWriteTool) Name() string {
	return "write_file"
}

func (f *FileWriteTool) Description() string {
	return "写入内容到文件"
}

func (f *FileWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "文件路径",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "文件内容",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (f *FileWriteTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	path := kwargs["path"].(string)
	content := kwargs["content"].(string)
	// 实际写入文件...
	return tool.TextResponse(fmt.Sprintf("已写入文件: %s (%d 字符)", path, len(content))), nil
}

// CodeExecuteTool 代码执行工具
type CodeExecuteTool struct{}

func (c *CodeExecuteTool) Name() string {
	return "execute_code"
}

func (c *CodeExecuteTool) Description() string {
	return "执行代码并返回结果"
}

func (c *CodeExecuteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "要执行的代码",
			},
			"language": map[string]any{
				"type":        "string",
				"description": "编程语言 (python, javascript, etc)",
			},
		},
		"required": []string{"code", "language"},
	}
}

func (c *CodeExecuteTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
	_ = kwargs["code"].(string)
	lang := kwargs["language"].(string)
	// actual execution...
	return tool.TextResponse(fmt.Sprintf("Executed %s code, output: ...", lang)), nil
}
