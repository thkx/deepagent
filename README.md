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
- **零重度依赖**：仅依赖 `go-openai`，不依赖过时的 langchaingo

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
