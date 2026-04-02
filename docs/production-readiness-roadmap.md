# DeepAgent 最小生产化改造清单（按优先级）

目标：在尽量少改动架构的前提下，把当前实现从“可运行”推进到“可上线并可维护”。

## 当前进度（2026-04-02）

- 已完成：P0-1（凭据外置）、P0-2（工具参数 schema 接入）、P0-3（HITL 拒绝阻断）、P0-5（不存在文件语义）
- 已完成（基础版）：P0-4（execute 安全边界收紧，已加语法/参数/目录/输出限制；容器级隔离仍在 P2）
- 已完成：P1-1（子代理文件名 slug + 去重）
- 已完成：P1-2（`ToolResult` 结构统一为 `ok/data/error/code`）
- 已完成（最小闭环）：P1-3（模型调用与运行摘要日志，含 request_id/thread_id、时延与错误率）
- 已完成：P1-4（新增 Invoke + ToolCall + Checkpoint + Resume 集成测试）
- 已完成：P2-1（Sandbox 容器隔离，支持 `process/docker/gvisor` 模式与资源/网络限制）
- 已完成（加强版）：P2-1（新增 rootless 检查、seccomp profile 策略、镜像白名单与 cosign 签名校验）
- 已完成（基础版）：P2-3（多模型适配：OpenAI / Anthropic / Ollama / Groq）

## P0（上线前必须）

1. 凭据与模型配置外置化
- 问题：当前默认 OpenAI key 是占位值，易误用。
- 改造：强制从 `env` 或注入 `opts.LLM` 获取；若缺失直接报错并给出可操作提示。
- 验收：示例在无 key 下可读错误，在有 key 下可直接运行。

2. 工具参数 schema 真正接入
- 问题：`Graph` 传给 LLM 的工具参数目前是 `nil`，限制 tool calling 准确性。
- 改造：让工具定义支持参数 schema，`convertToolsToLLMFormat` 传递真实 `Parameters`。
- 验收：至少 `write_file`、`edit_file`、`task`、`execute` 具备 required 字段约束。

3. HITL 审批结果必须生效
- 问题：当前 `WaitForApproval` 返回 reject 时，调用方没有阻断执行。
- 改造：若审批失败，跳过工具执行并将拒绝结果写回会话。
- 验收：配置了 HITL 的工具在 `n/reject` 时不执行。

4. `execute` 安全边界收紧
- 问题：当前是进程白名单 + `strings.Fields`，安全边界偏弱。
- 改造：增加参数级限制、工作目录限制、环境变量白名单、输出截断和审计日志。
- 验收：危险命令与越界访问可被稳定拦截，并有审计记录。

5. 文件系统语义对齐
- 问题：`read_file` 对不存在文件返回空字符串，调用方难区分“空文件”和“不存在”。
- 改造：不存在时返回显式错误（或在返回结构中增加 `exists` 字段）。
- 验收：上层提示与分支逻辑可准确处理文件不存在场景。

## P1（强烈建议，提升稳定性）

1. 子代理 merge 文件名安全化
- 问题：`subagent_<description>.md` 直接截断描述，可能含非法字符。
- 改造：统一 slug 化 + 长度限制 + 冲突去重。
- 验收：任意描述都可生成稳定、安全、可追踪的产物文件名。

2. 结构化输出与错误分类
- 问题：tool 执行结果多为字符串拼接，不利于自动处理。
- 改造：统一 `ToolResult` 结构（`ok/data/error/code`）。
- 验收：Graph 可基于错误码做重试、降级或终止。

3. 可观测性最小闭环
- 改造：增加 request_id/thread_id 贯穿日志，记录模型时延、tool 时延、错误率。
- 验收：可定位单次会话问题与热点工具。

4. 端到端测试补齐
- 改造：补充 Invoke+ToolCall+Checkpoint+Resume 的集成测试。
- 验收：关键路径回归可在 CI 一键发现。

## P2（演进项）

1. 真正隔离的 Sandbox（Docker/gVisor）
2. Checkpointer/Memory 接入 Redis 或 Postgres
3. 多模型适配（Anthropic/Ollama/Groq）
4. 更细粒度流式事件（token/tool state/todo state）
5. 中间件机制（规划、反思、后处理）

## 建议实施顺序（两周最小版本）

1. 第 1-2 天：P0-1/2（配置与 schema）
2. 第 3-4 天：P0-3/5（审批生效与文件语义）
3. 第 5-7 天：P0-4（execute 安全加固）+ 回归测试
4. 第 8-10 天：P1-1/2（结果结构化与子代理产物稳定）
5. 第 11-14 天：P1-3/4（观测与集成测试）
