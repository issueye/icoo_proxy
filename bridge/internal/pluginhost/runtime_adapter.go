package pluginhost

import (
	"time"

	"github.com/issueye/icoo_proxy/bridge/internal/service"
	"github.com/issueye/icoo_proxy/common/pluginipc"
)

// Ensure Manager satisfies service.PluginRuntime.
var _ service.PluginRuntime = (*Manager)(nil)

// List implements service.PluginRuntime.
func (m *Manager) List() []service.PluginRuntimeInstance {
	items := m.ListInstances()
	out := make([]service.PluginRuntimeInstance, 0, len(items))
	for _, inst := range items {
		view := service.PluginRuntimeInstance{
			ID:         inst.ID,
			Enabled:    inst.Entry.Enabled,
			Executable: inst.Entry.Executable,
			Status:     inst.Status,
			LastError:  inst.LastError,
			Endpoint:   inst.Endpoint,
		}
		if !inst.StartedAt.IsZero() {
			view.StartedAt = inst.StartedAt.Format(time.RFC3339)
		}
		if inst.Handshake != nil {
			view.PluginVersion = inst.Handshake.PluginVersion
			view.Capabilities = append([]string(nil), inst.Handshake.Capabilities...)
			view.SupportedIngress = append([]string(nil), inst.Handshake.SupportedIngress...)
			view.AdminBaseURL = inst.Handshake.AdminBaseURL
			if len(inst.Handshake.UIPages) > 0 {
				view.UIPages = append([]pluginipc.UIPage(nil), inst.Handshake.UIPages...)
			}
		}
		out = append(out, view)
	}
	return out
}

// ListInstances returns raw process instances including configured-but-stopped.
func (m *Manager) ListInstances() []*Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Instance, 0, len(m.plugins)+len(m.entries))
	seen := make(map[string]struct{}, len(m.plugins))
	for id, inst := range m.plugins {
		// Prefer live process state; refresh Entry from catalog when present.
		if e, ok := m.entries[id]; ok {
			cp := *inst
			cp.Entry = e
			out = append(out, &cp)
		} else {
			out = append(out, inst)
		}
		seen[id] = struct{}{}
	}
	for id, entry := range m.entries {
		if _, ok := seen[id]; ok {
			continue
		}
		out = append(out, &Instance{
			ID:     id,
			Entry:  entry,
			Status: "stopped",
		})
	}
	return out
}
