# plugin-grokbuild

Process plugin that adapts **Grok Build / SuperGrok** subscription traffic for `icoo_proxy`.

Inspired by community proxy designs such as [GreyGunG/grokbuild-proxy](https://github.com/GreyGunG/grokbuild-proxy) / [issueye/grokbuild-proxy](https://github.com/issueye/grokbuild-proxy).

> Unofficial. See [DISCLAIMER.md](./DISCLAIMER.md).

## What it does

| Capability | Behavior |
|------------|----------|
| IPC proxy | `proxy.complete` / `proxy.stream` for anthropic, openai-chat, openai-responses |
| Upstream | `cli-chat-proxy.grok.com/v1/responses` with CLI-like headers |
| Protocol | Converts via `common/ai_llm_proxy` to/from Responses |
| Models | Static catalog + optional live `/models` when token present |
| OAuth | Device Login (`auth.x.ai`) + refresh_token 轮转（singleflight） |
| Pre-refresh | 后台扫描即将过期凭据并提前刷新（默认 2 分钟一轮） |
| Billing | Admin 页拉取 `/billing` 与 `/billing?format=credits` |
| Desktop UI | Loopback admin UI；handshake 声明扩展页 |

## Build

```powershell
cd plugins/grokbuild
go build -o build/plugin-grokbuild.exe ./cmd/plugin-grokbuild
```

Or via monorepo packaging (also copies next to bridge/desktop bins):

```powershell
# from repo root
.\build-all.ps1
# or skip plugins: .\build-all.ps1 -SkipPlugins
```

Copy next to `bridge.exe` or set absolute `executable` in config.

## Configure bridge

```toml
[plugins.entries.grokbuild]
enabled = true
executable = "plugin-grokbuild.exe"
data_dir = ".data/plugins/grokbuild"
```

Restart bridge. Desktop sidebar **插件 → Grok 凭据** appears when the plugin is running.

## Provider routing

Create a provider:

- `vendor = "plugin"`
- `plugin_id = "grokbuild"`
- `base_url = "plugin://grokbuild"`
- `protocol` = preferred ingress (e.g. `anthropic`)

Add models (`grok-4`, `grok-4.5`, …) and route rules as usual.

## Credentials

Open the extension page (iframe via bridge `/api/v1/plugins/grokbuild/ui/`) and:

1. **Device Login**：浏览器打开 xAI 验证页，输入用户代码，成功后自动入库  
2. **手动添加** access / refresh token + 优先级  
3. **JSON 导入** 支持：
   - `{ "access_token": "..." }`
   - `{ "credentials": [ ... ] }`
   - `{ "accounts": { "id": { "accessToken": "..." } } }`
4. **额度**：用当前凭据探测 weekly/monthly billing（上游可用时）

### 多账号池

- 按 `priority` 降序选号，同优先级 round-robin  
- 非 2xx：`401/403/429/5xx` 冷却后可切换其它凭据（complete 最多 3 次；stream 仅 open 前 failover）  
- 连续 3 次 401/403 自动禁用该凭据  

Tokens stay under the plugin `data_dir` only (`credentials.json`, mode 0600).
