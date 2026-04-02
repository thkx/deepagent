# DeepAgent Go

**Go 语言实现的 Deep Agent 框架**，对外 API 与 LangChain Python `deepagents` 高度一致。

一个支持规划（write_todos）、虚拟文件系统、子代理递归委托、长时记忆、Human-in-the-loop 和安全代码执行的生产级 Agent 框架。

## 特性

- **API 一致性**：`CreateDeepAgent` + `Invoke`/`Stream` 与 Python 版几乎一致
- **完整文件系统**：`ls`、`read_file`、`write_file`、`edit_file`（支持 InMemory + Disk）
- **规划能力**：`write_todos` 工具（结构化输出）
- **子代理**：`task` 工具支持递归创建 + **结果自动 merge** 到父文件系统
- **长时记忆**：Memory Store（namespace + key-value）
- **持久化**：Checkpointer（thread_id 保存/恢复）
- **Skills 系统**：自动加载 `./skills/*.md` 并注入系统提示
- **Human-in-the-loop**：工具级中断与人工审批
- **安全执行**：`execute` 工具 + SandboxBackend
- **并发安全**：线程安全设计，支持高并发
- **零重度依赖**：仅依赖 `go-openai`，不依赖 langchaingo
- **多模型适配**：支持 `OpenAI / Anthropic / Ollama / Groq`

## 扩展文档

- [Invoke 时序图](./docs/invoke-sequence.md)
- [最小生产化改造清单](./docs/production-readiness-roadmap.md)

## 容器隔离 Sandbox（Docker/gVisor）

`execute` 工具现在支持三种隔离模式：

- `process`：进程白名单模式（默认）
- `docker`：通过 Docker 容器执行（`--network none`、资源限制、只读根文件系统）
- `gvisor`：通过 Docker + `runsc` 运行时执行（额外内核隔离）

可通过环境变量快速启用：

```bash
export DEEPAGENT_SANDBOX_MODE=docker   # 或 gvisor
export DEEPAGENT_SANDBOX_IMAGE=alpine:3.20
export DEEPAGENT_SANDBOX_RUNTIME=runsc # 仅 gvisor 模式使用
export DEEPAGENT_SANDBOX_ALLOWED_IMAGES=alpine:3.20,cgr.dev/*
export DEEPAGENT_SANDBOX_REQUIRE_ROOTLESS=true
export DEEPAGENT_SANDBOX_SECCOMP_PROFILE=./seccomp/default.json
export DEEPAGENT_SANDBOX_REQUIRE_SIGNED_IMAGES=true
export DEEPAGENT_SANDBOX_COSIGN_KEY=./cosign.pub
```

增强策略说明：
- 默认容器模式启用镜像白名单检查（未在 allowlist 的镜像会拒绝执行）
- 默认要求 rootless Docker（可通过 `DEEPAGENT_SANDBOX_REQUIRE_ROOTLESS=false` 放宽）
- 可强制要求 seccomp profile（`DEEPAGENT_SANDBOX_REQUIRE_SECCOMP=true`）
- 可启用 cosign 签名校验（`DEEPAGENT_SANDBOX_REQUIRE_SIGNED_IMAGES=true`）

也可通过 `Options.ExecuteConfig` 显式配置：

```go
cfg := builtin.DefaultExecuteConfig()
cfg.IsolationMode = "gvisor"
cfg.ContainerImage = "alpine:3.20"
cfg.ContainerRuntime = "runsc"

agt, err := agent.CreateDeepAgent(agent.Options{
    // ...
    ExecuteConfig: &cfg,
})
```

## 多模型适配（Anthropic/Ollama/Groq）

可通过 `Options.Provider` 选择：

- `openai`（默认）
- `anthropic`
- `groq`
- `ollama`

示例：

```go
agt, err := agent.CreateDeepAgent(agent.Options{
    Provider: "anthropic",
    Model:    "claude-3-5-sonnet-latest",
})
```

环境变量约定：

- `OPENAI_API_KEY`（openai）
- `ANTHROPIC_API_KEY`（anthropic）
- `GROQ_API_KEY`（groq）
- `OLLAMA_BASE_URL`（ollama，默认 `http://localhost:11434/v1`）
- `DEEPAGENT_PROVIDER`（全局默认 provider）

## 项目结构

```bash
deepagent/
├── agent/                  # 核心 Agent、Graph、Checkpointer、HITL、Subagent
│   ├── deepagent.go        # CreateDeepAgent 主入口
│   ├── graph.go            # ReAct + Planner 执行器
│   ├── types.go            # 核心类型（Input/Output/Message/State 等）
│   ├── options.go          # Options 定义
│   ├── checkpointer.go     # Checkpointer 接口与实现
│   ├── subagent.go         # 子代理工具（避免循环依赖）
│   └── hitl.go             # Human-in-the-loop
├── llms/
│   ├── llm.go              # LLM 抽象接口
│   ├── openai/             # OpenAI 实现（支持 tool calling + JSON mode）
│   │   └── openai.go
│   ├── anthropic/          # 未来可扩展
│   └── ollama/
├── tools/
│   ├── tool.go             # Tool 接口 + 工厂函数
│   └── builtin/
│       ├── write_todos.go
│       ├── skills.go       # Skills 加载
│       ├── execute.go      # execute 工具
│       └── fs/             # ls, read_file, write_file, edit_file + backend
│           ├── backend.go
│           ├── errors.go
│           ├── inmemory.go
│           ├── disk.go
│           ├── ls.go
│           ├── read.go
│           ├── write.go
│           └── edit.go
├── memory/
│   └── store.go            # 长时记忆 Store
├── examples/
│   └── research.go
└── README.md
```


### 已实现的功能（覆盖率 ≈ 80%）

类别 | 具体功能 | 状态 | 备注
|  ---- | ----  | ---- | ---- |
核心 API | CreateDeepAgent(Options) | 已实现 | 与 Python 版高度一致
核心 API | "DeepAgent.Invoke(ctx |  input)" | 已实现 | 支持 messages + thread_id
核心 API | "DeepAgent.Stream(ctx |  input)" | 已实现 | 支持 Event 通道
LLM 层 | OpenAI ChatModel + 基础 tool calling | 已实现 | llms/openai/openai.go
Tool 系统 | Tool 接口 + NewTool 工厂函数 | 已实现 | 标准工具封装
Tool Calling | 完整 Tool Call 解析、执行、结果反馈 | 已实现 | ReAct 循环中完整处理
规划工具 | write_todos | 已实现 | 返回结构化 Todo 列表
文件系统 | Backend 可插拔接口 | 已实现 | InMemory + Disk
文件系统工具 | ls、read_file、write_file、edit_file | 全部已实现 | 参数校验、安全路径
子代理 | task 工具（递归创建子代理） | 已实现 | 通过 agent/subagent.go
子代理 | 子代理结果自动 merge 到父文件系统 | 已实现 | 自动生成 subagent_xxx.md
持久化 | Checkpointer + FileCheckpointer | 已实现 | 支持 thread_id 恢复
长时记忆 | Memory Store（FileMemoryStore） | 已实现 | namespace + key-value
Skills 系统 | 自动加载 ./skills/*.md 并注入系统提示 | 已实现 | NewSkillsLoader
代码执行 | execute 工具 | 已实现 | 白名单基础安全版
Sandbox | SandboxBackend（实现完整 fs.Backend 接口） | 已实现 | 委托模式
Human-in-the-loop | 中断机制 + 人工审批 | 已实现 | InterruptConfig + WaitForApproval
执行器 | ReAct + Planner 多轮循环 + Checkpoint | 已实现 | 最多 25 轮，带反思
安全特性 | DiskBackend 路径安全检查（防目录遍历） | 已实现 | safePath
并发安全 | mutex / RWMutex | 已实现 | InMemoryBackend 安全

### 未实现的功能（待补充）
类别|具体功能|优先级|说明
|  ---- | ----  | ---- | ---- |
子代理|多层子代理树管理（parent-child 关系跟踪）|★★★★★|当前仅支持单层递归
子代理|子代理与父代理共享工具或部分上下文|★★★★|当前完全隔离
Memory|自动上下文总结与压缩（长上下文管理）|★★★★★|当前仅简单保存，未自动总结
Sandbox|更强隔离与策略增强（seccomp profile、镜像签名、rootless）|★★★★|已支持 Docker/gVisor 容器隔离，后续可继续强化
Human-in-the-loop|Web/API 版真实等待用户输入|★★★★★|当前仅控制台 fmt.Scanln
Human-in-the-loop|Tool Call 修改（modify）与重试机制|★★★★|当前仅 approve/reject
Tool|glob、grep、find、search 等高级 FS 工具|★★★★|可基于现有 Backend 快速添加
执行器|完整 LangGraph Pregel 风格状态图执行器|★★★★★|当前仍是简单 ReAct 循环
Structured Output|强 JSON mode + Schema（write_todos 等工具）|★★★★|当前 LLM 未强制返回 JSON
Observability|结构化日志、Tracing、Metrics|★★★|当前只有少量 fmt.Printf
多 LLM 支持|Anthropic、Groq、Ollama、Claude 等|★★★|llms/ 结构已准备好扩展
持久化|Redis / Postgres Checkpointer + Memory|★★★★|当前仅文件实现
CLI|命令行交互式 DeepAgent CLI|★★|可后续使用 cobra 添加
Middleware|自定义中间件系统（规划、反思、后处理）|★★★|当前无中间件机制
