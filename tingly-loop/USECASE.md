# Tingly-Loop Use Case

本文档展示一个完整的 tingly-loop 运行示例，包括中间过程和命令输出。

## 场景设定

假设我们有一个简单的 Go 项目，需要实现一个问候功能。项目结构如下：

```
myproject/
├── go.mod
├── main.go
└── (其他文件...)
```

---

## 方式一：交互式创建 Tasks（推荐新手）

### 1. 初始化 Tasks

```bash
$ cd myproject
$ tingly-loop init

🚀 Tingly-Loop Tasks Generator
This will create a tasks.json template for your project.

Project name: Greeter
Branch name [feature/greeter]:
Feature description (one line): Implement a greeting library with hello and goodbye functions

📝 Enter user stories (press Enter with empty input to finish):
   Format: <title> | <description>
   Example: Add login button | As a user, I want to see a login button

Story 1 (or press Enter to finish): Create hello function | As a developer, I need a Hello function

Story 2 (or press Enter to finish): Create goodbye function | As a developer, I need a Goodbye function

Story 3 (or press Enter to finish): Add unit tests | As a developer, I need tests for the greeter

Story 4 (or press Enter to finish):

✅ Created docs/loop/tasks.json with 3 stories

Next steps:
  1. Edit the file to refine acceptance criteria
  2. Run 'tingly-loop run' to start the loop
```

### 2. 编辑 Tasks（可选）

生成的 `docs/loop/tasks.json`：

```json
{
  "project": "Greeter",
  "branchName": "feature/greeter",
  "description": "Implement a greeting library with hello and goodbye functions",
  "userStories": [
    {
      "id": "US-001",
      "title": "Create hello function",
      "description": "As a developer, I need a Hello function",
      "acceptanceCriteria": [
        "Specific criterion 1",
        "Specific criterion 2",
        "Typecheck passes",
        "Tests pass"
      ],
      "priority": 1,
      "passes": false,
      "notes": ""
    }
  ]
}
```

你可以编辑文件来完善 acceptance criteria：

```json
{
  "id": "US-001",
  "title": "Create hello function",
  "description": "As a developer, I need a Hello function that greets users by name.",
  "acceptanceCriteria": [
    "Function named 'Hello' exists in greeter.go",
    "Function takes a name string and returns 'Hello, {name}!'",
    "Function handles empty name by returning 'Hello, World!'",
    "Typecheck passes",
    "Tests pass"
  ]
}
```

---

## 方式二：AI 生成 Tasks（推荐高级用户）

### 1. 从描述生成 Tasks

```bash
$ cd myproject
$ tingly-loop generate "Add user authentication with email and password login"

🤖 Generating tasks using claude agent...
Feature: Add user authentication with email and password login

{
  "project": "myproject",
  "branchName": "feature/user-authentication",
  "description": "Add user authentication with email and password login",
  "userStories": [
    {
      "id": "US-001",
      "title": "Add users table to database",
      "description": "As a developer, I need to store user credentials securely.",
      "acceptanceCriteria": [
        "Add users table with id, email, password_hash, created_at columns",
        "Email column has unique constraint",
        "Generate and run migration successfully",
        "Typecheck passes"
      ],
      "priority": 1,
      "passes": false,
      "notes": ""
    },
    {
      "id": "US-002",
      "title": "Create password hashing utility",
      "description": "As a developer, I need secure password hashing.",
      "acceptanceCriteria": [
        "Create auth/password.go with Hash and Verify functions",
        "Use bcrypt with appropriate cost factor",
        "Unit tests for both functions",
        "Typecheck passes"
      ],
      "priority": 2,
      "passes": false,
      "notes": ""
    },
    {
      "id": "US-003",
      "title": "Add signup API endpoint",
      "description": "As a user, I want to create an account with email and password.",
      "acceptanceCriteria": [
        "POST /api/auth/signup accepts email and password",
        "Returns 201 on success, 400 for invalid input, 409 for duplicate email",
        "Password is hashed before storage",
        "Typecheck passes"
      ],
      "priority": 3,
      "passes": false,
      "notes": ""
    },
    {
      "id": "US-004",
      "title": "Add login API endpoint",
      "description": "As a user, I want to log in with my email and password.",
      "acceptanceCriteria": [
        "POST /api/auth/login accepts email and password",
        "Returns 200 with session token on success, 401 for invalid credentials",
        "Typecheck passes"
      ],
      "priority": 4,
      "passes": false,
      "notes": ""
    },
    {
      "id": "US-005",
      "title": "Create login UI",
      "description": "As a user, I want a form to enter my credentials.",
      "acceptanceCriteria": [
        "Login form with email and password fields",
        "Form validation shows error messages",
        "Redirect to dashboard on successful login",
        "Typecheck passes",
        "Verify in browser"
      ],
      "priority": 5,
      "passes": false,
      "notes": ""
    }
  ]
}

✅ Created docs/loop/tasks.json

Next steps:
  1. Review and edit the generated tasks
  2. Run 'tingly-loop run' to start the loop
```

---

## 方式三：从现有文档导入（如果有 markdown 文档）

如果你已经有 markdown 格式的需求文档，可以用 AI 转换：

```bash
$ tingly-loop generate "$(cat docs/prd-auth.md)"
```

---

## 2. 查看初始状态

```bash
$ tingly-loop status

Project: Greeter
Branch: feature/greeter
Agent: claude
Description: Implement a greeting library with hello and goodbye functions

User Stories:
  [pending] US-001 (Priority 1): Create hello function
  [pending] US-002 (Priority 2): Create goodbye function
  [pending] US-003 (Priority 3): Add unit tests

Progress Log:
# Tingly Loop Progress Log
Started: 2026-02-14T22:10:00+08:00
---
```

---

## 3. 运行 Tingly-Loop

```bash
$ tingly-loop run --max-iterations 5

Starting Tingly Loop
Project: Greeter
Branch: feature/greeter
Agent: claude
Stories: 3 total, 0 completed

Switching to branch: feature/greeter

============================================================
  Iteration 1 of 5 (agent: claude)
============================================================

Next story: [US-001] Create hello function (Priority 1)

[Agent executes...]
✓ Created greeter.go with Hello function
✓ Tests pass
✓ Updated docs/loop/tasks.json (US-001: passes=true)

Iteration 1 complete. Progress: 1/3 stories

============================================================
  Iteration 2 of 5 (agent: claude)
============================================================

Next story: [US-002] Create goodbye function (Priority 2)

[Agent executes...]
✓ Added Goodbye function to greeter.go
✓ Tests pass
✓ Updated docs/loop/tasks.json (US-002: passes=true)

Iteration 2 complete. Progress: 2/3 stories

============================================================
  Iteration 3 of 5 (agent: claude)
============================================================

Next story: [US-003] Add unit tests (Priority 3)

[Agent executes...]
✓ Created greeter_test.go
✓ All tests pass with 100% coverage
✓ Updated docs/loop/tasks.json (US-003: passes=true)

All stories completed!

<promise>COMPLETE</promise>
Agent signaled completion at iteration 3
```

---

## 4. 最终状态

```bash
$ tingly-loop status

Project: Greeter
Branch: feature/greeter
Agent: claude
Description: Implement a greeting library with hello and goodbye functions

User Stories:
  [completed] US-001 (Priority 1): Create hello function
  [completed] US-002 (Priority 2): Create goodbye function
  [completed] US-003 (Priority 3): Add unit tests

Progress Log:
# Tingly Loop Progress Log
Started: 2026-02-14T22:10:00+08:00
---

## 2026-02-14 22:12:00 - US-001
- Implemented Hello function in greeter.go
- Files changed:
  - greeter.go
- **Learnings for future iterations:**
  - Use simple string concatenation for this project
---

## 2026-02-14 22:14:00 - US-002
- Implemented Goodbye function in greeter.go
- Files changed:
  - greeter.go
---

## 2026-02-14 22:16:00 - US-003
- Created comprehensive unit tests with 100% coverage
- Files changed:
  - greeter_test.go
- **Learnings for future iterations:**
  - Use table-driven tests for better organization
---
```

---

## 5. 项目最终文件

```
myproject/
├── go.mod
├── main.go
├── greeter.go              # Agent 创建
├── greeter_test.go         # Agent 创建
└── docs/
    └── loop/
        ├── tasks.json      # Agent 更新 (所有 passes: true)
        └── progress.md     # Agent 追加
```

---

## 6. 使用不同 Agent

### 使用 tingly-code 作为 Agent

```bash
$ tingly-loop run --agent tingly-code --config ../tingly-code/tingly-config.toml

Starting Tingly Loop
Project: Greeter
Branch: feature/greeter
Agent: tingly-code
Stories: 3 total, 0 completed

... (类似的迭代过程)
```

### 使用自定义二进制作为 Agent

```bash
$ tingly-loop run --agent subprocess --agent-binary ./my-custom-agent --agent-arg "--verbose"

Starting Tingly Loop
Project: Greeter
Branch: feature/greeter
Agent: my-custom-agent
Stories: 3 total, 0 completed

... (自定义 agent 的输出)
```

---

## 工作流对比

| 步骤 | Ralph | Tingly-Loop |
|------|-------|-------------|
| 1. 创建 Tasks | 使用 /prd skill 生成 markdown | `tingly-loop init` 交互式创建 |
| 2. 转换 Tasks | 使用 /ralph skill 转换为 JSON | `tingly-loop generate` AI 生成 JSON |
| 3. 运行循环 | `./ralph.sh` | `tingly-loop run` |
| 4. 查看状态 | `cat prd.json \| jq` | `tingly-loop status` |

---

## 关键点总结

1. **交互式初始化**: `tingly-loop init` 引导用户创建 tasks，无需手写 JSON
2. **AI 生成**: `tingly-loop generate` 从自然语言描述生成结构化 tasks
3. **循环控制**: tingly-loop 负责循环、状态管理、完成检测
4. **Agent 执行**: 实际工作由 agent 完成，拥有完整工具访问权限
5. **状态持久化**: tasks.json 记录任务状态，progress.md 记录学习积累
6. **完成信号**: Agent 输出 `<promise>COMPLETE</promise>` 表示所有任务完成
7. **迭代隔离**: 每次迭代都是独立的，但可以通过 progress.md 传递上下文

---

## 文件路径

| 文件 | 默认路径 |
|------|----------|
| Tasks 定义 | `docs/loop/tasks.json` |
| 进度日志 | `docs/loop/progress.md` |

可以通过 CLI 参数覆盖默认路径：
```bash
$ tingly-loop run --tasks ./my-tasks.json --progress ./my-progress.md
```
