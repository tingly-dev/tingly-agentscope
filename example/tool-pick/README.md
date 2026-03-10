# Tool-Pick Agent

智能工具选择 Agent，基于 AnyTool 的设计理念，为 tingly-agentscope 框架提供智能工具选择功能。

## 核心特性

1. **智能工具检索** - 多阶段工具选择策略
   - LLM 预过滤：将工具分类为实用工具和领域工具
   - 语义搜索：基于向量相似度进行工具匹配
   - 混合策略：结合 LLM 和语义搜索的优势

2. **质量感知排序** - 自我进化的工具质量追踪
   - 跟踪工具调用次数和成功率
   - 根据历史性能调整工具排名
   - 持久化质量数据以持续改进

3. **高效缓存** - 多层缓存机制
   - 向量缓存：持久化工具嵌入
   - 选择缓存：缓存选择结果
   - TTL 过期策略

## 架构设计

```
pkg/toolpick/
├── types.go              # 核心类型定义
├── toolpick.go           # ToolProvider 实现
├── selector/             # 选择策略
│   ├── selector.go       # 选择器接口
│   ├── semantic.go       # 语义选择器
│   ├── llm_filter.go     # LLM 过滤选择器
│   └── hybrid.go         # 混合选择器
├── ranking/              # 质量排序
│   └── quality.go        # 质量管理器
└── cache/                # 缓存层
    └── embedding.go      # 向量缓存
```

## 使用方法

### 基础用法

```go
import (
    "github.com/tingly-dev/tingly-agentscope/pkg/tool"
    "github.com/tingly-dev/tingly-agentscope/pkg/toolpick"
)

// 创建基础工具包
baseToolkit := tool.NewToolkit()

// 注册工具到不同分组
baseToolkit.CreateToolGroup("weather", "天气工具", true, "")
baseToolkit.Register(GetWeather{}, &tool.RegisterOptions{
    GroupName: "weather",
})

// 包装智能工具选择器
smartToolkit, err := toolpick.NewToolProvider(baseToolkit, &toolpick.Config{
    DefaultStrategy: "hybrid",
    MaxTools:       20,
    LLMThreshold:   50,
    EnableQuality:  true,
    QualityWeight:  0.2,
})
```

### 与 ReActAgent 集成

```go
import (
    "github.com/tingly-dev/tingly-agentscope/pkg/agent"
    "github.com/tingly-dev/tingly-agentscope/pkg/model/openai"
)

modelClient := openai.NewClient(&model.ChatModelConfig{
    ModelName: "gpt-4o",
    APIKey:    os.Getenv("OPENAI_API_KEY"),
})

reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
    Name:         "assistant",
    SystemPrompt: "你是一个有用的助手。",
    Model:        modelClient,
    Toolkit:      smartToolkit,  // 使用智能工具包
    Memory:       memory.NewHistory(100),
})
```

### 工具选择

```go
ctx := context.Background()

// 为特定任务选择工具
result, err := smartToolkit.SelectTools(ctx, "查询东京天气并计算15%小费", 10)

fmt.Printf("选择了 %d 个工具:\n", len(result.Tools))
for _, t := range result.Tools {
    fmt.Printf("  - %s: %s (score: %.3f)\n",
        t.Function.Name, t.Function.Description, result.Scores[t.Function.Name])
}

fmt.Printf("\n选择原因:\n%s\n", result.Reasoning)
```

### 质量报告

```go
// 获取工具质量报告
report := smartToolkit.GetQualityReport()

for name, record := range report {
    fmt.Printf("%s:\n", name)
    fmt.Printf("  调用次数: %d\n", record.CallCount)
    fmt.Printf("  成功率: %.2f%%\n", record.SuccessRate*100)
    fmt.Printf("  质量评分: %.3f\n", record.QualityScore())
}
```

## 选择策略

### 1. Semantic (语义搜索)

基于向量相似度进行工具匹配：
- 生成任务和工具的向量表示
- 计算余弦相似度
- 支持向量缓存以提升性能

### 2. LLM Filter (LLM 过滤)

使用 LLM 进行工具分类：
- 将工具分为实用工具和领域工具
- 精确选择需要的实用工具
- 包含所有相关领域的工具

### 3. Hybrid (混合策略)

结合两者优势：
- 工具数量多时使用 LLM 预过滤
- 对领域工具进行语义搜索
- 实用工具 + 顶级领域工具

## 配置选项

```go
type Config struct {
    // 选择器配置
    DefaultStrategy string  // "semantic", "llm_filter", "hybrid"
    MaxTools       int     // 默认返回的最大工具数
    LLMThreshold   int     // 使用 LLM 过滤的工具数量阈值

    // 质量配置
    EnableQuality  bool    // 启用质量追踪
    QualityWeight  float64 // 质量权重 (0.0-1.0)
    MinSuccessRate float64 // 最小成功率阈值

    // 缓存配置
    EnableCache    bool           // 启用缓存
    CacheDir       string         // 缓存目录
    CacheTTL       time.Duration  // 缓存过期时间

    // LLM 配置
    LLMModel       string         // LLM 模型名称
}
```

## 运行示例

```bash
cd example/tool-pick
go run ./cmd/tool-pick/main.go
```

## 设计理念

本实现基于以下原则：

1. **SOLID 原则**
   - 单一职责：每个组件专注于特定功能
   - 开闭原则：可扩展的选择策略系统
   - 接口隔离：最小化依赖

2. **DRY (Don't Repeat Yourself)**
   - 复用 tingly-agentscope 的工具系统
   - 统一的接口设计

3. **KISS (Keep It Simple, Stupid)**
   - 简单的向量嵌入实现
   - 清晰的代码结构

## 质量公式

```
final_score = semantic_score * (1 - quality_weight) + quality_score * quality_weight

quality_score = 0.6 * success_rate + 0.3 * description_quality + 0.1 * log_factor
```

- `success_rate`: 成功调用次数 / 总调用次数
- `description_quality`: LLM 评估的工具描述质量 (0-1)
- `log_factor`: 基于调用次数的对数因子

## 后续改进

- [ ] 集成实际的嵌入模型 API (OpenAI, Cohere)
- [ ] 实现真正的 LLM API 调用
- [ ] 添加更多选择策略
- [ ] 支持自定义嵌入模型
- [ ] 添加性能基准测试
