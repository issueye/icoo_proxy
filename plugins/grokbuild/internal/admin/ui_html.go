package admin

// indexHTML is the loopback admin UI (also embedded in Desktop via bridge UI proxy).
// Visual language aligns with icoo desktop UED: flat, compact, light neutral console.
const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Grok 凭据 · GrokBuild</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f3f4f6;
      --card: #ffffff;
      --border: #d4d7dc;
      --border-light: #e4e6ea;
      --text: #1f2329;
      --text-2: #4b5563;
      --muted: #6b7280;
      --primary: #2563eb;
      --primary-hover: #3b6ef0;
      --primary-soft: #eaf1ff;
      --success: #16a34a;
      --success-soft: #ecfaf0;
      --warning: #c2790a;
      --warning-soft: #fdf6e8;
      --danger: #dc2626;
      --danger-soft: #fdf0ef;
      --info-soft: #eaf1ff;
      --radius: 6px;
      --shadow: 0 1px 1px rgba(20, 22, 26, 0.04);
      --font: "Segoe UI", "PingFang SC", "Microsoft YaHei", Arial, sans-serif;
      --mono: "Cascadia Code", "SFMono-Regular", Consolas, monospace;
    }
    * { box-sizing: border-box; }
    html, body { height: 100%; }
    body {
      margin: 0;
      font-family: var(--font);
      font-size: 13px;
      line-height: 1.45;
      color: var(--text);
      background: var(--bg);
      padding: 14px 16px 24px;
    }
    a { color: var(--primary); text-decoration: none; }
    a:hover { text-decoration: underline; }

    .page-head {
      display: flex;
      flex-wrap: wrap;
      align-items: flex-start;
      justify-content: space-between;
      gap: 10px 16px;
      margin-bottom: 12px;
    }
    .page-head h1 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      letter-spacing: -0.01em;
    }
    .page-head .sub {
      margin: 4px 0 0;
      color: var(--muted);
      font-size: 12px;
      max-width: 52rem;
    }
    .badge-row { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      height: 22px;
      padding: 0 8px;
      border-radius: 999px;
      font-size: 11px;
      font-weight: 500;
      border: 1px solid var(--border);
      background: var(--card);
      color: var(--text-2);
      white-space: nowrap;
    }
    .badge.ok { background: var(--success-soft); border-color: #bbf7d0; color: var(--success); }
    .badge.warn { background: var(--warning-soft); border-color: #f5e0b8; color: var(--warning); }
    .badge.bad { background: var(--danger-soft); border-color: #fecaca; color: var(--danger); }
    .badge.info { background: var(--info-soft); border-color: #bfdbfe; color: var(--primary); }

    .stats {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 8px;
      margin-bottom: 12px;
    }
    @media (max-width: 720px) {
      .stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
    }
    .stat {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      box-shadow: var(--shadow);
      padding: 10px 12px;
      min-height: 58px;
    }
    .stat .k { font-size: 11px; color: var(--muted); margin-bottom: 4px; }
    .stat .v { font-size: 18px; font-weight: 600; letter-spacing: -0.02em; }
    .stat .h { font-size: 11px; color: var(--muted); margin-top: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

    .layout {
      display: grid;
      grid-template-columns: minmax(0, 1fr) minmax(0, 1.15fr);
      gap: 12px;
      align-items: start;
    }
    @media (max-width: 900px) {
      .layout { grid-template-columns: 1fr; }
    }

    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      box-shadow: var(--shadow);
      padding: 12px 14px;
    }
    .card-title {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
      margin-bottom: 10px;
    }
    .card-title h2 {
      margin: 0;
      font-size: 13px;
      font-weight: 600;
    }
    .card-title .hint { font-size: 11px; color: var(--muted); }

    .tabs {
      display: flex;
      gap: 4px;
      flex-wrap: wrap;
      margin-bottom: 12px;
      padding: 3px;
      background: #eef0f3;
      border-radius: var(--radius);
      border: 1px solid var(--border-light);
    }
    .tab {
      appearance: none;
      border: 0;
      background: transparent;
      color: var(--text-2);
      height: 28px;
      padding: 0 12px;
      border-radius: 4px;
      font-size: 12px;
      font-weight: 500;
      cursor: pointer;
      font-family: inherit;
    }
    .tab:hover { background: rgba(255,255,255,0.65); color: var(--text); }
    .tab.active {
      background: var(--card);
      color: var(--primary);
      box-shadow: 0 0 0 1px color-mix(in srgb, var(--primary) 25%, transparent);
    }

    label.field {
      display: block;
      font-size: 11px;
      font-weight: 500;
      color: var(--muted);
      margin: 0 0 4px;
    }
    .field-gap { margin-bottom: 10px; }
    input[type="text"], input[type="number"], input[type="url"], textarea {
      width: 100%;
      height: 30px;
      border: 1px solid var(--border);
      border-radius: 4px;
      background: #fff;
      color: var(--text);
      padding: 0 10px;
      font-size: 13px;
      font-family: inherit;
      outline: none;
    }
    textarea {
      height: auto;
      min-height: 64px;
      padding: 8px 10px;
      resize: vertical;
      font-family: var(--mono);
      font-size: 12px;
      line-height: 1.4;
    }
    input:focus, textarea:focus {
      border-color: var(--primary);
      box-shadow: 0 0 0 3px color-mix(in srgb, var(--primary) 14%, transparent);
    }
    input:disabled {
      background: #f4f5f7;
      color: var(--muted);
    }

    .row { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
    .row.spread { justify-content: space-between; }
    .stack { display: flex; flex-direction: column; gap: 10px; }

    button, .btn {
      appearance: none;
      border: 1px solid transparent;
      border-radius: 4px;
      height: 30px;
      padding: 0 12px;
      font-size: 12px;
      font-weight: 550;
      font-family: inherit;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 6px;
      background: var(--primary);
      color: #fff;
      white-space: nowrap;
    }
    button:hover { background: var(--primary-hover); }
    button:disabled { opacity: 0.55; cursor: not-allowed; }
    button.secondary {
      background: #fff;
      color: var(--text);
      border-color: var(--border);
    }
    button.secondary:hover { background: #f4f5f7; }
    button.ghost {
      background: transparent;
      color: var(--text-2);
      border-color: transparent;
    }
    button.ghost:hover { background: #eef0f3; color: var(--text); }
    button.danger {
      background: #fff;
      color: var(--danger);
      border-color: #fecaca;
    }
    button.danger:hover { background: var(--danger-soft); }
    button.xs { height: 24px; padding: 0 8px; font-size: 11px; }

    .alert {
      border-radius: var(--radius);
      border: 1px solid var(--border);
      padding: 8px 10px;
      font-size: 12px;
      line-height: 1.45;
      margin-bottom: 10px;
    }
    .alert.warn {
      background: var(--warning-soft);
      border-color: #f5e0b8;
      color: #8a5a08;
    }
    .alert.info {
      background: var(--info-soft);
      border-color: #bfdbfe;
      color: #1e40af;
    }
    .alert.ok {
      background: var(--success-soft);
      border-color: #bbf7d0;
      color: #166534;
    }

    .login-panel {
      border: 1px solid var(--border-light);
      background: #fafbfc;
      border-radius: var(--radius);
      padding: 12px;
    }
    .codebox {
      font-family: var(--mono);
      font-size: 26px;
      font-weight: 600;
      letter-spacing: 0.18em;
      padding: 14px 16px;
      background: #fff;
      border: 1px dashed #93c5fd;
      border-radius: var(--radius);
      color: var(--primary);
      text-align: center;
      user-select: all;
      margin: 8px 0;
    }
    .step {
      display: flex;
      gap: 8px;
      align-items: flex-start;
      font-size: 12px;
      color: var(--text-2);
      margin-top: 6px;
    }
    .step .n {
      flex: 0 0 18px;
      width: 18px;
      height: 18px;
      border-radius: 999px;
      background: var(--primary-soft);
      color: var(--primary);
      font-size: 11px;
      font-weight: 600;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      margin-top: 1px;
    }

    table.pool {
      width: 100%;
      border-collapse: collapse;
      font-size: 12px;
    }
    table.pool th {
      text-align: left;
      font-size: 11px;
      font-weight: 500;
      color: var(--muted);
      padding: 6px 8px;
      border-bottom: 1px solid var(--border);
      background: #fafbfc;
    }
    table.pool td {
      padding: 8px;
      border-bottom: 1px solid var(--border-light);
      vertical-align: middle;
    }
    table.pool tr:last-child td { border-bottom: 0; }
    table.pool tr:hover td { background: #fafbfc; }
    .name-cell { min-width: 120px; }
    .name-cell strong { display: block; font-weight: 600; font-size: 12px; }
    .name-cell .meta { color: var(--muted); font-size: 11px; margin-top: 2px; }
    .mono { font-family: var(--mono); font-size: 11px; color: var(--muted); }
    .pill {
      display: inline-flex;
      align-items: center;
      height: 20px;
      padding: 0 7px;
      border-radius: 999px;
      font-size: 11px;
      font-weight: 500;
      border: 1px solid var(--border);
      background: #fff;
      color: var(--text-2);
    }
    .pill.ok { background: var(--success-soft); border-color: #bbf7d0; color: var(--success); }
    .pill.warn { background: var(--warning-soft); border-color: #f5e0b8; color: var(--warning); }
    .pill.bad { background: var(--danger-soft); border-color: #fecaca; color: var(--danger); }

    .switch {
      position: relative;
      width: 34px;
      height: 18px;
      display: inline-block;
    }
    .switch input { opacity: 0; width: 0; height: 0; }
    .switch span {
      position: absolute;
      inset: 0;
      background: #c4c8cf;
      border-radius: 999px;
      cursor: pointer;
      transition: 0.15s;
    }
    .switch span::before {
      content: "";
      position: absolute;
      width: 14px;
      height: 14px;
      left: 2px;
      top: 2px;
      background: #fff;
      border-radius: 50%;
      transition: 0.15s;
      box-shadow: 0 1px 2px rgba(0,0,0,0.15);
    }
    .switch input:checked + span { background: var(--primary); }
    .switch input:checked + span::before { transform: translateX(16px); }

    .empty {
      text-align: center;
      padding: 28px 12px;
      color: var(--muted);
      font-size: 12px;
    }
    .empty strong { display: block; color: var(--text); font-size: 13px; margin-bottom: 4px; }

    .billing-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 8px;
      margin-bottom: 10px;
    }
    @media (max-width: 560px) {
      .billing-grid { grid-template-columns: 1fr; }
    }
    .bill-card {
      border: 1px solid var(--border-light);
      border-radius: var(--radius);
      padding: 10px;
      background: #fafbfc;
    }
    .bill-card .t { font-size: 11px; color: var(--muted); margin-bottom: 4px; }
    .bill-card .n { font-size: 16px; font-weight: 600; }
    pre.raw {
      margin: 0;
      white-space: pre-wrap;
      word-break: break-word;
      background: #f4f5f7;
      border: 1px solid var(--border-light);
      border-radius: 4px;
      padding: 10px;
      font-family: var(--mono);
      font-size: 11px;
      max-height: 260px;
      overflow: auto;
      color: var(--text-2);
    }

    .toast-host {
      position: fixed;
      right: 14px;
      bottom: 14px;
      z-index: 50;
      display: flex;
      flex-direction: column;
      gap: 6px;
      pointer-events: none;
    }
    .toast {
      pointer-events: auto;
      min-width: 220px;
      max-width: 360px;
      background: #111827;
      color: #f9fafb;
      border-radius: 6px;
      padding: 10px 12px;
      font-size: 12px;
      box-shadow: 0 8px 24px rgba(15, 23, 42, 0.18);
      animation: toast-in 0.18s ease-out;
    }
    .toast.err { background: #7f1d1d; }
    .toast.ok { background: #14532d; }
    @keyframes toast-in {
      from { opacity: 0; transform: translateY(6px); }
      to { opacity: 1; transform: none; }
    }

    .prio-input {
      width: 64px;
      height: 24px !important;
      padding: 0 6px !important;
      font-size: 12px !important;
      text-align: center;
    }
    .actions-cell { white-space: nowrap; }
    .split { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
    @media (max-width: 560px) { .split { grid-template-columns: 1fr; } }
    .muted { color: var(--muted); font-size: 12px; }
    .hr { border: 0; border-top: 1px solid var(--border-light); margin: 12px 0; }
  </style>
</head>
<body>
  <header class="page-head">
    <div>
      <h1>GrokBuild / SuperGrok</h1>
      <p class="sub">管理 Device Login 与多账号凭据池。令牌仅保存在本机插件 data_dir，不会写入 bridge 数据库。</p>
    </div>
    <div class="badge-row" id="head-badges">
      <span class="badge info">非官方插件</span>
      <span class="badge" id="badge-proxy">代理 · -</span>
    </div>
  </header>

  <div class="alert warn">
    风险自负：非官方能力，可能违反第三方 ToS。OAuth 仅连接 <strong>https://auth.x.ai</strong>；上游为 Grok Build CLI 协议。
  </div>

  <section class="stats" id="stats">
    <div class="stat"><div class="k">凭据总数</div><div class="v" id="stat-total">-</div><div class="h">池中全部账号</div></div>
    <div class="stat"><div class="k">可用</div><div class="v" id="stat-active">-</div><div class="h">已启用且有 Token</div></div>
    <div class="stat"><div class="k">冷却 / 异常</div><div class="v" id="stat-cool">-</div><div class="h">临时不可用</div></div>
    <div class="stat"><div class="k">出站代理</div><div class="v" id="stat-proxy" style="font-size:13px">-</div><div class="h" id="stat-proxy-hint">环境 / 直连</div></div>
  </section>

  <div class="layout">
    <section class="card">
      <div class="card-title">
        <h2>操作</h2>
        <span class="hint">登录 · 导入 · 额度 · 网络</span>
      </div>
      <div class="tabs" role="tablist">
        <button class="tab active" type="button" data-tab="login">Device Login</button>
        <button class="tab" type="button" data-tab="manual">手动 / 导入</button>
        <button class="tab" type="button" data-tab="billing">额度</button>
        <button class="tab" type="button" data-tab="network">代理</button>
      </div>

      <div id="panel-login">
        <div class="login-panel">
          <p class="muted" style="margin:0 0 8px">使用 xAI 设备码登录。浏览器打开验证页并输入代码后，凭据会自动写入池中。</p>
          <div class="row">
            <button id="start-login" type="button">开始 Device Login</button>
            <button class="secondary" id="copy-code" type="button" disabled>复制用户码</button>
            <button class="secondary" id="open-verify" type="button" disabled>打开验证页</button>
          </div>
          <div id="login-box" style="display:none;margin-top:12px">
            <div class="muted">用户代码</div>
            <div class="codebox" id="user-code">----</div>
            <div class="step"><span class="n">1</span><span>复制上方代码，或直接点击「打开验证页」</span></div>
            <div class="step"><span class="n">2</span><span id="verify-uri" class="mono">-</span></div>
            <div class="step"><span class="n">3</span><span id="login-status">等待授权…</span></div>
          </div>
        </div>
      </div>

      <div id="panel-manual" style="display:none">
        <div class="split">
          <div>
            <div class="field-gap">
              <label class="field" for="label">显示名称</label>
              <input id="label" type="text" value="SuperGrok" />
            </div>
            <div class="field-gap">
              <label class="field" for="priority">优先级（越大越优先）</label>
              <input id="priority" type="number" value="0" />
            </div>
            <div class="field-gap">
              <label class="field" for="token">Access Token</label>
              <textarea id="token" rows="3" placeholder="粘贴 access_token"></textarea>
            </div>
            <div class="field-gap">
              <label class="field" for="refresh">Refresh Token（可选）</label>
              <textarea id="refresh" rows="2" placeholder="可选，用于自动续期"></textarea>
            </div>
            <button id="save" type="button">保存到凭据池</button>
          </div>
          <div>
            <div class="field-gap">
              <label class="field" for="import-json">JSON 导入</label>
              <textarea id="import-json" rows="10" placeholder='{"access_token":"..."} 或 {"credentials":[...]} 或 {"accounts":{...}}'></textarea>
            </div>
            <button id="import-btn" type="button" class="secondary">导入 JSON</button>
            <p class="muted" style="margin:8px 0 0">支持单 token、credentials 数组、accounts 映射等常见导出格式。</p>
          </div>
        </div>
      </div>

      <div id="panel-billing" style="display:none">
        <div class="row spread" style="margin-bottom:10px">
          <p class="muted" style="margin:0">使用当前选中凭据探测 weekly / monthly 额度（上游可用时）。</p>
          <button id="load-billing" type="button">刷新额度</button>
        </div>
        <div class="billing-grid" id="billing-cards" style="display:none">
          <div class="bill-card"><div class="t">凭据</div><div class="n" id="bill-cred">-</div></div>
          <div class="bill-card"><div class="t">状态</div><div class="n" id="bill-status">-</div></div>
        </div>
        <pre class="raw" id="billing-out">点击「刷新额度」加载…</pre>
      </div>

      <div id="panel-network" style="display:none">
        <p class="muted" style="margin:0 0 10px">访问 auth.x.ai / cli-chat-proxy.grok.com 的出站代理。支持 HTTP 与 SOCKS5。留空则使用环境变量或直连。</p>
        <div class="field-gap">
          <label class="field" for="http-proxy">代理 URL</label>
          <input id="http-proxy" type="url" placeholder="http://127.0.0.1:7890 或 socks5://127.0.0.1:7891" />
        </div>
        <p class="muted" id="proxy-effective">当前生效：-</p>
        <div class="row" style="margin-top:8px">
          <button id="save-proxy" type="button">保存并生效</button>
          <button class="secondary" id="test-proxy" type="button">测试连通性</button>
          <button class="ghost" id="clear-proxy" type="button">清空</button>
        </div>
        <pre class="raw" id="proxy-test-out" style="margin-top:10px">尚未测试</pre>
      </div>
    </section>

    <section class="card">
      <div class="card-title">
        <h2>凭据池</h2>
        <div class="row">
          <button class="secondary xs" id="reload" type="button">刷新</button>
        </div>
      </div>
      <div class="alert info" style="margin-bottom:10px">
        调度规则：优先级降序 → 同优先级 round-robin；401/403/429/5xx 可冷却切换；连续 3 次鉴权失败自动禁用。
      </div>
      <div style="overflow:auto">
        <table class="pool">
          <thead>
            <tr>
              <th style="width:18%">启用</th>
              <th>账号</th>
              <th style="width:12%">优先级</th>
              <th style="width:14%">状态</th>
              <th style="width:12%">令牌</th>
              <th style="width:18%">失败</th>
              <th style="width:10%"></th>
            </tr>
          </thead>
          <tbody id="list"></tbody>
        </table>
        <div class="empty" id="empty-pool" style="display:none">
          <strong>还没有凭据</strong>
          使用左侧 Device Login，或手动粘贴 / 导入 JSON。
        </div>
      </div>
    </section>
  </div>

  <div class="toast-host" id="toasts"></div>

  <script>
    let pollTimer = null;
    let lastUserCode = '';
    let lastVerifyURI = '';
    let credentialsCache = [];

    const API_BASE = (() => {
      const path = location.pathname || '';
      const marker = '/ui';
      const i = path.lastIndexOf(marker);
      if (i < 0) return '';
      return path.slice(0, i + marker.length).replace(/\/+$/, '');
    })();
    function api(path) {
      const p = path.startsWith('/') ? path : '/' + path;
      return API_BASE + p;
    }

    function esc(s) {
      return String(s || '').replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
      }[c]));
    }

    function toast(text, kind) {
      const host = document.getElementById('toasts');
      const el = document.createElement('div');
      el.className = 'toast' + (kind === 'err' ? ' err' : kind === 'ok' ? ' ok' : '');
      el.textContent = text;
      host.appendChild(el);
      setTimeout(() => {
        el.style.opacity = '0';
        el.style.transition = 'opacity .2s';
        setTimeout(() => el.remove(), 220);
      }, 3200);
    }

    async function readError(res) {
      const text = await res.text();
      try {
        const j = JSON.parse(text);
        return j.error || j.message || text;
      } catch {
        return text || res.statusText || 'request failed';
      }
    }

    function showTab(tab) {
      document.querySelectorAll('.tab').forEach(b => {
        b.classList.toggle('active', b.dataset.tab === tab);
      });
      document.getElementById('panel-login').style.display = tab === 'login' ? '' : 'none';
      document.getElementById('panel-manual').style.display = tab === 'manual' ? '' : 'none';
      document.getElementById('panel-billing').style.display = tab === 'billing' ? '' : 'none';
      document.getElementById('panel-network').style.display = tab === 'network' ? '' : 'none';
      if (tab === 'network') loadSettings();
      if (tab === 'billing') { /* keep last */ }
    }

    document.querySelectorAll('.tab').forEach(btn => {
      btn.onclick = () => showTab(btn.dataset.tab);
    });

    function statusPill(c) {
      if (!c.enabled) return '<span class="pill bad">禁用</span>';
      if (c.cooling) return '<span class="pill warn">冷却</span>';
      if (c.has_token) return '<span class="pill ok">可用</span>';
      return '<span class="pill bad">无令牌</span>';
    }

    function formatExpiry(iso) {
      if (!iso) return '';
      try {
        const d = new Date(iso);
        if (isNaN(d.getTime()) || d.getFullYear() < 2000) return '';
        return d.toLocaleString();
      } catch { return ''; }
    }

    function updateStats(list, health) {
      const total = list.length;
      let active = 0, cool = 0;
      list.forEach(c => {
        if (c.cooling || (!c.enabled && c.failure_count > 0)) cool++;
        if (c.enabled && c.has_token && !c.cooling) active++;
      });
      document.getElementById('stat-total').textContent = String(total);
      document.getElementById('stat-active').textContent = String(active);
      document.getElementById('stat-cool').textContent = String(cool);
      if (health) {
        const proxy = health.http_proxy_effective || health.http_proxy || '(直连/环境)';
        document.getElementById('stat-proxy').textContent = proxy.length > 28 ? proxy.slice(0, 28) + '…' : proxy;
        document.getElementById('stat-proxy').title = proxy;
        document.getElementById('stat-proxy-hint').textContent = health.http_proxy ? 'settings.json' : '环境 / 直连';
        const badge = document.getElementById('badge-proxy');
        badge.textContent = '代理 · ' + (health.http_proxy ? '已配置' : '默认');
        badge.className = 'badge' + (health.http_proxy ? ' info' : '');
      }
    }

    async function loadHealth() {
      try {
        const res = await fetch(api('/api/health'));
        if (!res.ok) return null;
        return await res.json();
      } catch { return null; }
    }

    async function load() {
      try {
        const [credRes, health] = await Promise.all([
          fetch(api('/api/credentials')),
          loadHealth()
        ]);
        if (!credRes.ok) throw new Error(await readError(credRes));
        const data = await credRes.json();
        const list = data.credentials || [];
        credentialsCache = list;
        updateStats(list, health);

        const tb = document.getElementById('list');
        const empty = document.getElementById('empty-pool');
        tb.innerHTML = '';
        if (!list.length) {
          empty.style.display = '';
          return;
        }
        empty.style.display = 'none';
        list.forEach(c => {
          const tr = document.createElement('tr');
          const exp = formatExpiry(c.expires_at);
          const tok = (c.has_token ? 'AT' : '—') + (c.has_refresh ? '+RT' : '');
          const fail = (c.failure_count || 0) + (c.last_error ? '<div class="meta" title="' + esc(c.last_error) + '">' + esc(String(c.last_error).slice(0, 48)) + '</div>' : '');
          tr.innerHTML =
            '<td><label class="switch" title="启用/禁用"><input type="checkbox" data-act="toggle" data-id="' + esc(c.id) + '"' + (c.enabled ? ' checked' : '') + ' /><span></span></label></td>' +
            '<td class="name-cell"><strong>' + esc(c.label || c.id) + '</strong>' +
              (c.email ? '<div class="meta">' + esc(c.email) + '</div>' : '') +
              (exp ? '<div class="meta">到期 ' + esc(exp) + '</div>' : '') +
              '<div class="meta mono">' + esc(c.id) + '</div></td>' +
            '<td><input class="prio-input" type="number" data-act="prio" data-id="' + esc(c.id) + '" value="' + Number(c.priority || 0) + '" /></td>' +
            '<td>' + statusPill(c) + '</td>' +
            '<td class="mono">' + tok + '</td>' +
            '<td class="muted">' + fail + '</td>' +
            '<td class="actions-cell"><button class="danger xs" type="button" data-act="del" data-id="' + esc(c.id) + '">删除</button></td>';
          tb.appendChild(tr);
        });

        tb.querySelectorAll('[data-act="toggle"]').forEach(el => {
          el.onchange = async () => {
            const id = el.dataset.id;
            const cur = credentialsCache.find(x => x.id === id);
            try {
              const res = await fetch(api('/api/credentials'), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  id: id,
                  label: cur && cur.label || '',
                  priority: cur ? Number(cur.priority || 0) : 0,
                  enabled: !!el.checked
                })
              });
              if (!res.ok) throw new Error(await readError(res));
              toast(el.checked ? '已启用' : '已禁用', 'ok');
              load();
            } catch (e) {
              toast(String(e.message || e), 'err');
              el.checked = !el.checked;
            }
          };
        });
        tb.querySelectorAll('[data-act="prio"]').forEach(el => {
          el.onchange = async () => {
            const id = el.dataset.id;
            const cur = credentialsCache.find(x => x.id === id);
            try {
              const res = await fetch(api('/api/credentials'), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  id: id,
                  label: cur && cur.label || '',
                  priority: Number(el.value || 0),
                  enabled: cur ? !!cur.enabled : true
                })
              });
              if (!res.ok) throw new Error(await readError(res));
              toast('优先级已更新', 'ok');
              load();
            } catch (e) {
              toast(String(e.message || e), 'err');
              load();
            }
          };
        });
        tb.querySelectorAll('[data-act="del"]').forEach(btn => {
          btn.onclick = async () => {
            if (!confirm('确定删除该凭据？此操作不可恢复。')) return;
            try {
              const res = await fetch(api('/api/credentials?id=' + encodeURIComponent(btn.dataset.id)), { method: 'DELETE' });
              if (!res.ok) throw new Error(await readError(res));
              toast('已删除', 'ok');
              load();
            } catch (e) {
              toast(String(e.message || e), 'err');
            }
          };
        });
      } catch (e) {
        toast('加载凭据失败：' + (e.message || e), 'err');
      }
    }

    async function loadSettings() {
      try {
        const res = await fetch(api('/api/settings'));
        const data = await res.json();
        document.getElementById('http-proxy').value = data.http_proxy || '';
        document.getElementById('proxy-effective').textContent = '当前生效：' + (data.http_proxy_effective || '-');
      } catch (e) {
        document.getElementById('proxy-effective').textContent = '加载失败：' + e;
      }
    }

    document.getElementById('save-proxy').onclick = async () => {
      const http_proxy = document.getElementById('http-proxy').value.trim();
      try {
        const res = await fetch(api('/api/settings'), {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ http_proxy })
        });
        if (!res.ok) throw new Error(await readError(res));
        const data = await res.json();
        toast('代理已保存并立即生效', 'ok');
        document.getElementById('proxy-effective').textContent = '当前生效：' + (data.http_proxy_effective || http_proxy || '(none)');
        load();
      } catch (e) {
        toast(String(e.message || e), 'err');
      }
    };
    document.getElementById('clear-proxy').onclick = () => {
      document.getElementById('http-proxy').value = '';
      document.getElementById('save-proxy').click();
    };
    document.getElementById('test-proxy').onclick = async () => {
      const el = document.getElementById('proxy-test-out');
      el.textContent = '测试中…';
      try {
        const http_proxy = document.getElementById('http-proxy').value.trim();
        const res = await fetch(api('/api/settings/proxy-test'), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ http_proxy })
        });
        const text = await res.text();
        try {
          const j = JSON.parse(text);
          el.textContent = JSON.stringify(j, null, 2);
          toast(j.ok ? '连通性正常' : ('连通失败：' + (j.error || j.status)), j.ok ? 'ok' : 'err');
        } catch {
          el.textContent = text;
        }
      } catch (e) {
        el.textContent = String(e);
        toast(String(e.message || e), 'err');
      }
    };

    document.getElementById('start-login').onclick = async () => {
      const btn = document.getElementById('start-login');
      btn.disabled = true;
      if (pollTimer) clearInterval(pollTimer);
      try {
        toast('正在请求设备码…');
        const res = await fetch(api('/api/oauth/device/start'), { method: 'POST' });
        if (!res.ok) throw new Error(await readError(res));
        const sess = await res.json();
        lastUserCode = sess.user_code || '';
        lastVerifyURI = sess.verification_uri_complete || sess.verification_uri || '';
        document.getElementById('login-box').style.display = '';
        document.getElementById('user-code').textContent = lastUserCode || '----';
        document.getElementById('verify-uri').textContent = lastVerifyURI || '（无验证链接）';
        document.getElementById('login-status').textContent = '状态: ' + (sess.status || 'pending') + ' · 等待浏览器授权';
        document.getElementById('copy-code').disabled = !lastUserCode;
        document.getElementById('open-verify').disabled = !lastVerifyURI;
        toast('请在浏览器完成授权', 'ok');
        pollTimer = setInterval(async () => {
          try {
            const r = await fetch(api('/api/oauth/device/status?id=' + encodeURIComponent(sess.id)));
            if (!r.ok) return;
            const st = await r.json();
            document.getElementById('login-status').textContent =
              '状态: ' + st.status + (st.error ? ' · ' + st.error : '');
            if (st.status === 'success' || st.status === 'error' || st.status === 'expired') {
              clearInterval(pollTimer);
              pollTimer = null;
              if (st.status === 'success') {
                toast('登录成功，凭据已写入', 'ok');
                load();
              } else {
                toast('登录结束：' + (st.error || st.status), 'err');
              }
            }
          } catch { /* ignore poll errors */ }
        }, 2500);
      } catch (e) {
        toast(String(e.message || e), 'err');
      } finally {
        btn.disabled = false;
      }
    };

    document.getElementById('copy-code').onclick = async () => {
      if (!lastUserCode) return;
      try {
        await navigator.clipboard.writeText(lastUserCode);
        toast('用户码已复制', 'ok');
      } catch {
        toast('复制失败，请手动选择代码', 'err');
      }
    };
    document.getElementById('open-verify').onclick = () => {
      if (lastVerifyURI) window.open(lastVerifyURI, '_blank', 'noopener');
    };

    document.getElementById('save').onclick = async () => {
      const body = {
        label: document.getElementById('label').value,
        access_token: document.getElementById('token').value,
        refresh_token: document.getElementById('refresh').value,
        priority: Number(document.getElementById('priority').value || 0),
        enabled: true
      };
      if (!String(body.access_token || '').trim()) {
        toast('请填写 Access Token', 'err');
        return;
      }
      try {
        const res = await fetch(api('/api/credentials'), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body)
        });
        if (!res.ok) throw new Error(await readError(res));
        document.getElementById('token').value = '';
        document.getElementById('refresh').value = '';
        toast('已保存到凭据池', 'ok');
        load();
      } catch (e) {
        toast(String(e.message || e), 'err');
      }
    };

    document.getElementById('import-btn').onclick = async () => {
      const raw = document.getElementById('import-json').value;
      if (!String(raw || '').trim()) {
        toast('请粘贴 JSON', 'err');
        return;
      }
      try {
        const res = await fetch(api('/api/credentials/import'), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ raw: raw, label: 'imported' })
        });
        if (!res.ok) throw new Error(await readError(res));
        const data = await res.json();
        toast('已导入 ' + (data.imported || 0) + ' 条', 'ok');
        document.getElementById('import-json').value = '';
        load();
      } catch (e) {
        toast(String(e.message || e), 'err');
      }
    };

    function pickBillingNumber(obj, keys) {
      if (!obj || typeof obj !== 'object') return null;
      for (const k of keys) {
        if (obj[k] != null && obj[k] !== '') return obj[k];
      }
      // shallow search one level
      for (const v of Object.values(obj)) {
        if (v && typeof v === 'object' && !Array.isArray(v)) {
          for (const k of keys) {
            if (v[k] != null && v[k] !== '') return v[k];
          }
        }
      }
      return null;
    }

    document.getElementById('load-billing').onclick = async () => {
      const el = document.getElementById('billing-out');
      const cards = document.getElementById('billing-cards');
      el.textContent = '加载中…';
      cards.style.display = 'none';
      try {
        const res = await fetch(api('/api/billing'));
        const text = await res.text();
        if (!res.ok) throw new Error(text || res.statusText);
        let data = {};
        try { data = JSON.parse(text); } catch { el.textContent = text; return; }
        el.textContent = JSON.stringify(data, null, 2);
        document.getElementById('bill-cred').textContent = data.label || data.credential_id || '-';
        const weekly = data.weekly;
        const monthly = data.monthly;
        const wLeft = pickBillingNumber(weekly, ['remaining', 'remaining_credits', 'credits_remaining', 'left', 'balance']);
        const mLeft = pickBillingNumber(monthly, ['remaining', 'remaining_credits', 'credits_remaining', 'left', 'balance']);
        let statusLine = '';
        if (data.weekly_status != null) statusLine += 'weekly HTTP ' + data.weekly_status + '  ';
        if (data.monthly_status != null) statusLine += 'monthly HTTP ' + data.monthly_status;
        if (wLeft != null || mLeft != null) {
          statusLine = (wLeft != null ? '周额度 ' + wLeft + '  ' : '') + (mLeft != null ? '月额度 ' + mLeft : '');
        }
        if (data.weekly_error) statusLine = data.weekly_error;
        if (data.monthly_error) statusLine = (statusLine ? statusLine + ' · ' : '') + data.monthly_error;
        document.getElementById('bill-status').textContent = statusLine || '已返回 JSON';
        cards.style.display = '';
        toast('额度已刷新', 'ok');
      } catch (e) {
        el.textContent = String(e.message || e);
        toast(String(e.message || e), 'err');
      }
    };

    document.getElementById('reload').onclick = () => { load(); toast('已刷新'); };
    load();
  </script>
</body>
</html>
`
