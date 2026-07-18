# icoo_proxy 文档索引

本目录是 monorepo 的权威文档入口。模块路径、协议契约、插件 IPC 与发布元数据以这里和根目录 README 为准；历史设计/计划文档可能仍含旧目录名，阅读时请先看文首状态标记。

## 快速入口

| 文档 | 说明 |
|------|------|
| [../README.md](../README.md) | 英文产品说明、构建与使用 |
| [../README.cn.md](../README.cn.md) | 中文产品说明、构建与使用 |
| [workspace.md](./workspace.md) | Go workspace 模块边界与分层 |
| [openapi.yaml](./openapi.yaml) | 管理 API 契约（Bridge Management API） |
| [plugin-ipc-contract.md](./plugin-ipc-contract.md) | 进程插件 IPC 线协议（Frozen v1） |
| [plugin-ipc-sdk.md](./plugin-ipc-sdk.md) | `common/pluginipc` Client/Server SDK 指南 |
| [../CHANGELOG.md](../CHANGELOG.md) | 发布变更 |
| [../VERSION](../VERSION) | 单一版本源（当前 2.0.1） |

## 模块 README

| 模块 | 文档 |
|------|------|
| `common/` | [../common/README.md](../common/README.md) |
| `common/ai_llm_proxy/` | [../common/ai_llm_proxy/README.md](../common/ai_llm_proxy/README.md) |
| `bridge/` | [../bridge/README.md](../bridge/README.md) |
| `desktop/` | [../desktop/README.md](../desktop/README.md) |
| `plugins/` | [../plugins/README.md](../plugins/README.md) |
| `plugins/mock/` | [../plugins/mock/README.md](../plugins/mock/README.md) |
| `plugins/grokbuild/` | [../plugins/grokbuild/README.md](../plugins/grokbuild/README.md) |

## 设计文档（`design/`）

| 文档 | 状态 | 说明 |
|------|------|------|
| [design/2026-07-16-process-plugin-architecture.md](./design/2026-07-16-process-plugin-architecture.md) | 已实现基线 | 进程插件架构全文 |
| [design/2026-07-16-process-plugin-architecture-summary.md](./design/2026-07-16-process-plugin-architecture-summary.md) | 已实现基线 | 架构摘要；文中旧路径见迁移表 |
| [design/2026-07-16-plugin-extension-pages.md](./design/2026-07-16-plugin-extension-pages.md) | 已实现基线 | 插件扩展页（Desktop iframe + Bridge 反代） |

## 计划文档（`plans/`）

| 文档 | 状态 | 说明 |
|------|------|------|
| [plans/2026-07-10-priority-bugfixes.md](./plans/2026-07-10-priority-bugfixes.md) | 已完成（历史） | 访问控制、密钥、健康检查等 |
| [plans/2026-07-11-protocol-converter-bugfixes.md](./plans/2026-07-11-protocol-converter-bugfixes.md) | 部分完成（历史） | 转换器修复；路径已迁到 `common/ai_llm_proxy` |
| [plans/2026-07-16-development-plan.md](./plans/2026-07-16-development-plan.md) | 已完成（历史） | P1–P10 工程强化 |
| [plans/2026-07-16-process-plugin-development-plan.md](./plans/2026-07-16-process-plugin-development-plan.md) | 已实现基线 | 进程插件落地计划 |
| [plans/2026-07-16-usage-accounting-contract.md](./plans/2026-07-16-usage-accounting-contract.md) | 生效中 | usage / cache token 语义约定 |

## 报告与归档（`report/`）

| 文档 | 状态 | 说明 |
|------|------|------|
| [report/项目分析报告.md](./report/项目分析报告.md) | 当前快照 | 2026-07-18 结构/模块/风险分析 |
| [../desktop/docs/项目分析报告.md](../desktop/docs/项目分析报告.md) | 历史归档 | 桌面侧早期分析，路径已过时 |
| [../desktop/docs/desktop-ui-optimization-design.md](../desktop/docs/desktop-ui-optimization-design.md) | 设计参考 | UI 优化方向，非实现清单 |
| [../bridge/docs/](../bridge/docs/) | 历史重构文档 | `icoo_server` → `icoo_llm_bridge` 时期设计，仅供溯源 |

## 路径迁移速查

旧文档中常见目录名已统一为：

| 旧路径 | 当前路径 |
|--------|----------|
| `icoo_llm_bridge/` | `bridge/` |
| `icoo_desktop/` | `desktop/` |
| `bridge/internal/utils/ai_llm_proxy` | `common/ai_llm_proxy` |
| `bridge/pkg/pluginipc` | `common/pluginipc` |
| `bridge/internal/constants` | `common/constants` |
| `bridge/internal/model/domain` | `common/domain` |
| `bridge/internal/utils/idgen` | `common/idgen` |
| `bridge/internal/view` | `common/view` |

## 文档维护约定

1. **用户入口**：根 `README.md` / `README.cn.md` 保持双语同步。
2. **模块边界**：以 `docs/workspace.md` 与各模块 README 为准；插件禁止 import `bridge/internal/...`。
3. **契约冻结**：`plugin-ipc-contract.md` 与 `openapi.yaml` 变更需同步实现与测试。
4. **历史文档**：不再维护正文细节时，在文首加状态标记，并指向当前权威文档，避免静默误导。
5. **版本**：发布版本只读根目录 `VERSION`；OpenAPI `info.version` 与之对齐。
