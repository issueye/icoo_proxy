# Design Summary: Modular Process-Plugin Architecture + GrokBuild Plugin

**Design doc:** `C:\Users\issue\AppData\Local\Temp\grok-issue\grok-design-doc-04bd1385.md` (**Rev 3**)  
**Review file:** `C:\Users\issue\AppData\Local\Temp\grok-issue\grok-design-review-04bd1385.md` (Issues 1–18 addressed; 0 open)  
**Date:** 2026-07-16  
**Status:** Draft Rev 3 — **approved for Phase 0**  

## What was produced

A full design (Chinese body, English identifiers) for process-level plugins in `icoo_proxy`, with GrokBuild as the first optional plugin. **Rev 2** closed Issues 1–15 (module graph, framing, proxy branch, admin, Job/PGID, MVP-A/B, PR plan). **Rev 3** closed Issues 16–18: atomic raw-followup under multiplexed RPC, Anthropic/sticky header allowlist, stream.open non-2xx without SSE commit.

## Codebase inputs reviewed

- **Bridge:** container, proxy_service, route_resolver, provider/vendor, traffic_record, config loader, Check/Chat/FetchModels
- **Desktop:** app.go single-level child process management
- **grokbuild-proxy:** DESIGN.md, package boundaries, Go 1.26.5

## Key architectural choices (Rev 2)

1. **Host = `icoo_llm_bridge`**; Desktop only starts/stops bridge.
2. **Process plugins + IPC:** go-winio Named Pipe / Unix pathname UDS; plugin **Listen**, host **Dial**.
3. **Module graph (KD-14):** SDK at `icoo_llm_bridge/pkg/pluginipc` (same module); plugin `icoo_plugin_grokbuild` + `replace`; **no root go.work in v1**.
4. **Framing:** length-prefix JSON-RPC 2.0 + **`raw-followup`** via atomic **`WriteMessage`** (write mutex across JSON+RAW) + demux **`expect_raw_body`**; `max_frame_bytes` follows **64 MiB**.
5. **Streaming:** open-result-before-chunk; **open non-2xx ⇒ no SSE**; write mutex; max **32** streams; sticky headers allowlisted (`anthropic-version`/`beta`, session ids); usage → traffic.
6. **Proxy branch:** **before** `ConvertRequest`; raw downstream body; skip double conversion; `OnlyStream` enforced.
7. **Routing:** `Vendor=plugin` + `plugin_id`; `Provider.Protocol` = UI preferred ingress only; traffic `UpstreamProtocol=plugin:<id>`.
8. **Admin:** Check→health, FetchModels→models.list, Chat→400 (v1); Desktop form later.
9. **Process tree:** Windows Job Object `KILL_ON_JOB_CLOSE` + Unix PGID.
10. **Shutdown:** `Server.Shutdown` drain → plugin.shutdown → Kill.
11. **GrokBuild:** Hybrid C main path; **HTTP sidecar B permanent dual path**; **MVP-A** (no tools/thinking) then **MVP-B**.
12. **Security:** SDDL `D:P(A;;GA;;;OW)`; env token + residual same-user risk; private admin default off.

## Document sections

Overview, Background, Goals, **Key Decisions KD-1…19**, Design (IPC, host, proxy, admin, Grok), API mapping table, Data model, Alternatives, Security, Observability, Rollout, Risks, Open Questions (non-blocking), Development Plan, **PR Plan PR-00…PR-14**.

## Implementation order (compressed)

1. PR-00/01/02 — contract + `pkg/pluginipc` + transports  
2. PR-03/04 — config + pluginhost (Job/PGID) + DI  
3. PR-05/06/07 — Admin plugins, provider plugin vendor, proxy branch  
4. PR-08/09/10/11 — grok skeleton → core → **credentials** → MVP-A proxy  
5. PR-12 packaging; PR-13 optional UI; PR-14 MVP-B  

## Non-negotiables

- Do not break existing HTTP provider paths  
- Loopback-first; IPC ACL + host_token  
- GrokBuild optional, default off, disclaimer  
- v1 = one real plugin (MVP-A), not a marketplace  
