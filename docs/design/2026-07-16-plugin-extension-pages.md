# 插件扩展页（Desktop Extension Pages）

## 问题

SuperGrok / GrokBuild 等插件需要独立管理凭据、账号池、账单诊断等 UI。桌面壳不应为每个插件硬编码页面。

## 方案

```text
plugin process
  ├─ IPC (proxy / models / health)
  └─ loopback HTTP admin UI  (127.0.0.1:random)
           ▲
           │ reverse proxy (only loopback targets)
bridge  /api/v1/plugins/:id/ui/*
           ▲
desktop  sidebar ──► /#/ext/:pluginId/:pageId  ──iframe──► bridge UI proxy
```

### Handshake 声明

```json
{
  "admin_base_url": "http://127.0.0.1:52341",
  "ui_pages": [
    {
      "id": "credentials",
      "title": "Grok 凭据",
      "path": "/",
      "icon": "key",
      "group": "插件",
      "description": "SuperGrok 令牌管理"
    }
  ]
}
```

### Bridge API

| Method | Path | 用途 |
|--------|------|------|
| GET | `/api/v1/plugins/ui-pages` | 汇总运行中插件的扩展页 |
| ANY | `/api/v1/plugins/:id/ui/*` | 反代到插件 loopback UI |

### Desktop

- 轮询 `ui-pages`，动态合并到侧栏分组
- 路由 `/ext/:pluginId/:pageId` → `PluginExtensionView` iframe
- 不直接访问插件随机端口（统一走 bridge，便于鉴权与同源策略）

## 安全

- 仅允许 `admin_base_url` 为 loopback
- 反代剥离 `Authorization` / `X-Api-Key`，避免把 bridge 管理密钥泄漏给插件
- 本地默认 `allow_local_without_auth` 时 iframe 可加载；非本机需管理鉴权
