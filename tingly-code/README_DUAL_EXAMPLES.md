# Dual Act Agent 示例说明

本目录包含两个 Dual Act Agent 的执行示例，展示了不同复杂度的任务场景。

## 示例文件

### 1. `output.dual.md` - 简单任务示例

**任务**: 创建一个简单的 Go 函数计算两个数的和

**特点**:
- 单轮 H-R 交互
- R 代理直接完成任务
- H 代理评估后决定 CONTINUE（要求验证代码）
- 任务较简单，未触发多轮迭代

**适用场景**: 快速原型、简单功能实现

---

### 2. `output.dual.complex.md` - 复杂多步骤任务示例

**任务**: 实现一个线程安全的通用栈，支持 Push、Pop、Peek、IsEmpty、Size 操作，需要包含完整的单元测试覆盖边界情况（空栈、单元素、并发访问）

**特点**:
- **多轮 H-R 交互**（超过 10 轮工具调用）
- **自我纠错**: R 代理自动修复缺少的 `time` 包导入
- **完整测试覆盖**: 包含 18 个测试用例，100% 代码覆盖率
- **性能基准测试**: 包含 6 个 benchmark 测试
- **竞态检测**: 通过 `go test -race` 验证
- **实际应用示例**: 创建了 README 和 example/main.go

**输出统计**:
- 1218 行交互日志
- 包含以下文件:
  - `stack.go` - 线程安全的栈实现（约 100 行）
  - `stack_test.go` - 完整测试套件（约 650 行）
  - `README.md` - 使用文档
  - `example/main.go` - 使用示例

**测试结果**:
```
=== RUN   TestNewStack
--- PASS: TestNewStack (0.00s)
=== RUN   TestPushAndPop
--- PASS: TestPushAndPop (0.00s)
=== RUN   TestPeek
--- PASS: TestPeek (0.00s)
=== RUN   TestIsEmpty
--- PASS: TestIsEmpty (0.00s)
=== RUN   TestSize
--- PASS: TestSize (0.00s)
=== RUN   TestPopEmptyStack
--- PASS: TestPopEmptyStack (0.00s)
=== RUN   TestSingleElement
--- PASS: TestSingleElement (0.00s)
=== RUN   TestGenericTypes
--- PASS: TestGenericTypes (0.00s)
=== RUN   TestClear
--- PASS: TestClear (0.00s)
=== RUN   TestToSlice
--- PASS: TestToSlice (0.00s)
=== RUN   TestConcurrentPush
--- PASS: TestConcurrentPush (0.01s)
=== RUN   TestConcurrentPop
--- PASS: TestConcurrentPop (0.01s)
=== RUN   TestConcurrentPushAndPop
--- PASS: TestConcurrentPushAndPop (0.00s)
=== RUN   TestConcurrentPeek
--- PASS: TestConcurrentPeek (0.00s)
=== RUN   TestConcurrentIsEmpty
--- PASS: TestConcurrentIsEmpty (0.00s)
=== RUN   TestConcurrentSize
--- PASS: TestConcurrentSize (0.00s)
=== RUN   TestRaceConditionPushPop
--- PASS: TestRaceConditionPushPop (0.00s)
=== RUN   TestStressTest
--- PASS: TestStressTest (0.50s)
PASS
coverage: 100.0% of statements
ok  	example/tingly-code	1.553s
```

**性能基准**:
```
BenchmarkPush-2             	18722542	        61.01 ns/op	      41 B/op	       0 allocs/op
BenchmarkPop-2              	35040770	        33.94 ns/op	       0 B/op	       0 allocs/op
BenchmarkPeek-2             	70070484	        17.82 ns/op	       0 B/op	       0 allocs/op
BenchmarkConcurrentPush-2   	17957095	        56.77 ns/op	      42 B/op	       0 allocs/op
BenchmarkConcurrentPop-2    	26784050	        45.71 ns/op	       0 B/op	       0 allocs/op
```

---

## H-R 交互模式对比

| 特性 | 简单任务 | 复杂任务 |
|------|--------------------------|-------------------------------------|
| 任务复杂度 | 低（单函数） | 高（完整数据结构 + 测试 + 文档） |
| H-R 循环次数 | 1 次 | 10+ 次工具调用 |
| 自我纠错 | 不需要 | 需要（修复导入、参数类型等） |
| 代码质量验证 | 基础 | 完整（单元测试 + 竞态检测 + benchmark） |
| 输出文件 | 1 个源文件 | 4 个文件（源码 + 测试 + 文档 + 示例） |
| 文档生成 | 无 | README + 示例代码 |

---

## 如何使用

### 运行简单任务
```bash
TINGLY_CONFIG=tingly-config.toml ./cmd/tingly-code/tingly-code dual "创建一个 Go 函数计算两数之和"
```

### 运行复杂任务
```bash
TINGLY_CONFIG=tingly-config.toml ./cmd/tingly-code/tingly-code dual "实现线程安全的泛型栈，包含完整测试和文档"
```

### 配置文件示例
```toml
[agent]
  name = "tingly"
  [agent.model]
    model_type = "anthropic"
    model_name = "tingly/cc"
    api_key = "your-api-key"
    base_url = "http://localhost:12580/tingly/claude_code"
    temperature = 0.3
    max_tokens = 8000

[dual]
  enabled = true
  max_hr_loops = 5  # H-R 最大交互次数
  # [dual.human] 可选 - 不指定则复用 [agent] 配置
```

---

## 关键要点

1. **任务复杂度决定交互轮次**: 简单任务可能只需 1 轮，复杂任务需要多轮迭代
2. **H 代理的评估至关重要**: Planner 会发现 R 代理遗漏的问题（如缺少导入、参数类型错误等）
3. **R 代理具备自我纠错能力**: 根据 H 的反馈，R 会主动修复问题
4. **测试驱动验证**: 复杂任务中，运行测试是验证代码正确性的关键步骤

---

## 总结

这两个示例展示了 Dual Act Agent 在不同复杂度任务上的表现：

- **简单任务**: 高效直接，适合快速原型
- **复杂任务**: 系统性强，能处理完整软件开发生命周期（编码 → 测试 → 文档）

选择合适的模式取决于你的任务需求和期望的代码质量。
