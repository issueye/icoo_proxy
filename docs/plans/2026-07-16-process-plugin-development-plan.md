# Process Plugin Development Plan

> Date: 2026-07-16  
> Design: [docs/design/2026-07-16-process-plugin-architecture.md](../design/2026-07-16-process-plugin-architecture.md)  
> Contract: [docs/plugin-ipc-contract.md](../plugin-ipc-contract.md)

## Goal

Add process-level plugins to `icoo_proxy` via IPC (Windows Named Pipe / Unix domain socket), with GrokBuild as the first optional plugin. Existing HTTP provider paths must remain unchanged.

## Phases

### Phase 0 — IPC SDK (PR-00 … PR-02)

- [x] Design + contract docs
- [x] `pkg/pluginipc` framing, JSON-RPC, raw-followup, Client/Server
- [x] Windows / Unix transports
- **Accept:** `go test ./pkg/pluginipc/...` ✅

### Phase 1 — Host lifecycle (PR-03 … PR-04)

- [x] Config `plugins` section
- [x] `pluginhost.Manager` (spawn, Job/PGID, handshake, heartbeat)
- [x] Container Start/Shutdown order
- [x] `cmd/mockplugin` + host integration test
- **Accept:** mock plugin healthy ✅

### Phase 2 — Admin + routing (PR-05 … PR-07)

- [x] Admin `/api/v1/plugins*`
- [x] `Vendor=plugin` + `plugin_id`
- [x] Proxy branch before ConvertRequest
- [x] Check / FetchModels plugin branches; Chat → 400 for plugins
- **Accept:** unit tests for plugin complete path; full suite green ✅

### Phase 3 — GrokBuild MVP-A (PR-08 … PR-11)

- [ ] Plugin skeleton
- [ ] Port auth/storage/lb/upstream/executor
- [ ] Credentials import/login
- [ ] Real proxy + models.list
- **Accept:** MVP-A matrix; default disabled; disclaimer

### Phase 4 — Packaging (PR-12)

- [ ] build-all optional plugin binary
- [ ] README / DISCLAIMER / path-B docs

### Phase 5 — MVP-B (PR-14)

- [ ] tools + thinking parity

## Engineering rules

1. Do not break existing provider proxy paths.
2. Plugin default off (`enabled=false`).
3. No `go.work` in v1; plugin uses `replace`.
4. Tests for concurrent raw-followup + ping body integrity.
5. Open-result-before-chunk is mandatory.
