package admin

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>GrokBuild / SuperGrok</title>
  <style>
    :root { color-scheme: light dark; font-family: ui-sans-serif, system-ui, sans-serif; }
    body { margin: 0; padding: 20px; background: #0f1419; color: #e7ecf3; }
    h1 { font-size: 18px; margin: 0 0 8px; }
    h2 { font-size: 15px; margin: 0 0 12px; }
    .sub { color: #9aa7b8; font-size: 13px; margin-bottom: 16px; line-height: 1.5; }
    .card { background: #1a222d; border: 1px solid #2a3544; border-radius: 12px; padding: 16px; margin-bottom: 16px; }
    label { display: block; font-size: 12px; color: #9aa7b8; margin-bottom: 6px; }
    input, textarea { width: 100%; box-sizing: border-box; border-radius: 8px; border: 1px solid #334155; background: #0f1419; color: #e7ecf3; padding: 10px 12px; margin-bottom: 12px; }
    button { background: #3b82f6; color: white; border: 0; border-radius: 8px; padding: 10px 14px; cursor: pointer; font-weight: 600; }
    button.secondary { background: #334155; }
    button.danger { background: #b91c1c; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    th, td { text-align: left; padding: 8px 6px; border-bottom: 1px solid #2a3544; vertical-align: top; }
    .warn { background: #3b2f12; border: 1px solid #854d0e; color: #fde68a; border-radius: 8px; padding: 10px 12px; font-size: 12px; margin-bottom: 16px; }
    .row { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
    .ok { color: #4ade80; }
    .bad { color: #f87171; }
    .muted { color: #9aa7b8; font-size: 12px; }
    .tabs { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
    .tab { background: #243041; }
    .tab.active { background: #3b82f6; }
    .codebox { font-family: ui-monospace, monospace; font-size: 22px; letter-spacing: 2px; padding: 12px; background: #0f1419; border-radius: 8px; border: 1px dashed #475569; display: inline-block; margin: 8px 0; }
    pre { white-space: pre-wrap; word-break: break-word; background: #0f1419; padding: 12px; border-radius: 8px; font-size: 12px; max-height: 240px; overflow: auto; }
  </style>
</head>
<body>
  <h1>GrokBuild / SuperGrok</h1>
  <p class="sub">非官方插件：Device Login · 多账号池 · refresh 轮转 · 额度探测。令牌仅保存在本机插件 data_dir。</p>
  <div class="warn">风险自负：可能违反第三方 ToS。OAuth 仅连接 https://auth.x.ai 。</div>

  <div class="card">
    <div class="tabs">
      <button class="tab active" data-tab="login" type="button">Device Login</button>
      <button class="tab" data-tab="manual" type="button">手动/导入</button>
      <button class="tab" data-tab="billing" type="button">额度</button>
      <button class="tab" data-tab="network" type="button">代理</button>
    </div>

    <div id="panel-login">
      <p class="sub">使用 xAI 设备码登录（浏览器打开链接并输入代码）。登录成功后自动写入凭据池。</p>
      <div class="row">
        <button id="start-login" type="button">开始 Device Login</button>
        <button class="secondary" id="reload" type="button">刷新凭据列表</button>
      </div>
      <div id="login-box" style="display:none;margin-top:12px">
        <div>用户代码</div>
        <div class="codebox" id="user-code">----</div>
        <div class="muted" id="verify-uri"></div>
        <div class="row" style="margin-top:8px">
          <a id="verify-link" href="#" target="_blank" rel="noopener" style="color:#93c5fd">在浏览器打开验证页</a>
        </div>
        <p class="sub" id="login-status">等待授权…</p>
      </div>
      <p id="msg" class="sub"></p>
    </div>

    <div id="panel-manual" style="display:none">
      <label>显示名称</label>
      <input id="label" value="SuperGrok" />
      <label>优先级</label>
      <input id="priority" type="number" value="0" />
      <label>Access Token</label>
      <textarea id="token" rows="2"></textarea>
      <label>Refresh Token（可选）</label>
      <textarea id="refresh" rows="2"></textarea>
      <div class="row">
        <button id="save" type="button">保存</button>
      </div>
      <hr style="border-color:#2a3544;margin:16px 0" />
      <label>JSON 导入</label>
      <textarea id="import-json" rows="6" placeholder='{"access_token":"...","refresh_token":"..."}'></textarea>
      <div class="row">
        <button id="import-btn" type="button">导入</button>
      </div>
    </div>

    <div id="panel-billing" style="display:none">
      <div class="row">
        <button id="load-billing" type="button">拉取额度</button>
      </div>
      <pre id="billing-out" class="muted">点击拉取额度…</pre>
    </div>

    <div id="panel-network" style="display:none">
      <p class="sub">配置访问 auth.x.ai / cli-chat-proxy.grok.com 的出站代理。支持 HTTP 与 SOCKS5。留空则使用系统环境变量 HTTP_PROXY / HTTPS_PROXY，或直连。</p>
      <label>代理 URL</label>
      <input id="http-proxy" placeholder="http://127.0.0.1:7890 或 socks5://127.0.0.1:7891" />
      <p class="muted" id="proxy-effective">当前生效：-</p>
      <div class="row">
        <button id="save-proxy" type="button">保存代理</button>
        <button class="secondary" id="test-proxy" type="button">测试连通性</button>
        <button class="secondary" id="clear-proxy" type="button">清空（环境/直连）</button>
      </div>
      <pre id="proxy-test-out" class="muted" style="margin-top:12px">尚未测试</pre>
    </div>
  </div>

  <div class="card">
    <h2>凭据池</h2>
    <table>
      <thead><tr><th>名称</th><th>优先级</th><th>状态</th><th>Token</th><th>失败</th><th></th></tr></thead>
      <tbody id="list"></tbody>
    </table>
  </div>

  <script>
    const msg = document.getElementById('msg');
    let pollTimer = null;

    // When embedded via bridge reverse-proxy the page lives under
    //   /api/v1/plugins/<id>/ui/
    // Absolute "/api/..." would hit the bridge (NOT_FOUND). Prefix with the
    // embed base; when opened on the plugin's loopback admin directly, base is "".
    const API_BASE = (() => {
      const path = location.pathname || '';
      const marker = '/ui';
      const i = path.lastIndexOf(marker);
      if (i < 0) return '';
      let base = path.slice(0, i + marker.length);
      // Drop anything after /ui (e.g. /ui/index.html)
      return base.replace(/\/+$/, '');
    })();
    function api(path) {
      const p = path.startsWith('/') ? path : '/' + path;
      return API_BASE + p;
    }

    document.querySelectorAll('.tab').forEach(btn => {
      btn.onclick = () => {
        document.querySelectorAll('.tab').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        const tab = btn.dataset.tab;
        document.getElementById('panel-login').style.display = tab === 'login' ? '' : 'none';
        document.getElementById('panel-manual').style.display = tab === 'manual' ? '' : 'none';
        document.getElementById('panel-billing').style.display = tab === 'billing' ? '' : 'none';
        document.getElementById('panel-network').style.display = tab === 'network' ? '' : 'none';
        if (tab === 'network') loadSettings();
      };
    });

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
      const res = await fetch(api('/api/settings'), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ http_proxy })
      });
      const text = await res.text();
      if (!res.ok) { msg.textContent = text; return; }
      let data = {};
      try { data = JSON.parse(text); } catch {}
      msg.textContent = '代理已保存并立即生效';
      document.getElementById('proxy-effective').textContent = '当前生效：' + (data.http_proxy_effective || http_proxy || '(none)');
    };
    document.getElementById('clear-proxy').onclick = async () => {
      document.getElementById('http-proxy').value = '';
      document.getElementById('save-proxy').click();
    };
    document.getElementById('test-proxy').onclick = async () => {
      const el = document.getElementById('proxy-test-out');
      el.textContent = '测试中…';
      const http_proxy = document.getElementById('http-proxy').value.trim();
      const res = await fetch(api('/api/settings/proxy-test'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ http_proxy })
      });
      const text = await res.text();
      try { el.textContent = JSON.stringify(JSON.parse(text), null, 2); }
      catch { el.textContent = text; }
    };

    async function load() {
      const res = await fetch(api('/api/credentials'));
      const data = await res.json();
      const tb = document.getElementById('list');
      tb.innerHTML = '';
      (data.credentials || []).forEach(c => {
        const status = (!c.enabled) ? '<span class="bad">禁用</span>'
          : (c.cooling ? '<span class="bad">冷却</span>'
          : (c.has_token ? '<span class="ok">可用</span>' : '<span class="bad">无令牌</span>'));
        const tok = (c.has_token ? 'AT' : '-') + (c.has_refresh ? '+RT' : '');
        const tr = document.createElement('tr');
        tr.innerHTML = '<td>' + esc(c.label||c.id) + (c.email ? '<div class="muted">'+esc(c.email)+'</div>' : '') + '</td>' +
          '<td>' + (c.priority||0) + '</td><td>' + status + '</td><td class="muted">' + tok + '</td>' +
          '<td class="muted">' + (c.failure_count||0) + (c.last_error ? '<div>'+esc(c.last_error)+'</div>' : '') + '</td>' +
          '<td><button class="danger" data-id="'+esc(c.id)+'">删除</button></td>';
        tb.appendChild(tr);
      });
      tb.querySelectorAll('button.danger').forEach(btn => {
        btn.onclick = async () => {
          await fetch(api('/api/credentials?id=' + encodeURIComponent(btn.dataset.id)), { method: 'DELETE' });
          load();
        };
      });
    }

    document.getElementById('start-login').onclick = async () => {
      msg.textContent = '请求设备码…';
      if (pollTimer) clearInterval(pollTimer);
      const res = await fetch(api('/api/oauth/device/start'), { method: 'POST' });
      if (!res.ok) { msg.textContent = await res.text(); return; }
      const sess = await res.json();
      document.getElementById('login-box').style.display = '';
      document.getElementById('user-code').textContent = sess.user_code || '----';
      const uri = sess.verification_uri_complete || sess.verification_uri || '';
      document.getElementById('verify-uri').textContent = uri;
      const link = document.getElementById('verify-link');
      link.href = uri || '#';
      document.getElementById('login-status').textContent = '状态: ' + (sess.status || 'pending');
      msg.textContent = '请在浏览器完成授权';
      pollTimer = setInterval(async () => {
        const r = await fetch(api('/api/oauth/device/status?id=' + encodeURIComponent(sess.id)));
        if (!r.ok) return;
        const st = await r.json();
        document.getElementById('login-status').textContent = '状态: ' + st.status + (st.error ? ' · ' + st.error : '');
        if (st.status === 'success' || st.status === 'error' || st.status === 'expired') {
          clearInterval(pollTimer);
          if (st.status === 'success') {
            msg.textContent = '登录成功，凭据已写入: ' + (st.credential_id || '');
            load();
          }
        }
      }, 2500);
    };

    document.getElementById('save').onclick = async () => {
      const body = {
        label: document.getElementById('label').value,
        access_token: document.getElementById('token').value,
        refresh_token: document.getElementById('refresh').value,
        priority: Number(document.getElementById('priority').value || 0),
        enabled: true
      };
      const res = await fetch(api('/api/credentials'), { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) });
      msg.textContent = res.ok ? '已保存' : await res.text();
      if (res.ok) { document.getElementById('token').value=''; document.getElementById('refresh').value=''; load(); }
    };
    document.getElementById('import-btn').onclick = async () => {
      const body = { raw: document.getElementById('import-json').value, label: 'imported' };
      const res = await fetch(api('/api/credentials/import'), { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) });
      if (!res.ok) { msg.textContent = await res.text(); return; }
      const data = await res.json();
      msg.textContent = '已导入 ' + (data.imported||0) + ' 条';
      load();
    };
    document.getElementById('load-billing').onclick = async () => {
      const el = document.getElementById('billing-out');
      el.textContent = '加载中…';
      const res = await fetch(api('/api/billing'));
      const text = await res.text();
      try { el.textContent = JSON.stringify(JSON.parse(text), null, 2); }
      catch { el.textContent = text; }
    };
    document.getElementById('reload').onclick = load;
    function esc(s){ return String(s||'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])); }
    load();
  </script>
</body>
</html>
`
