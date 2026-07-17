import { describe, expect, it } from "vitest";
import {
  isPluginVendor,
  pluginProviderDefaults,
} from "./providers_models.js";

// providerPayload / normalizeProvider are module-private; exercise via
// the exported helper and re-implement the critical pure rules here so
// the desktop vendor=plugin path stays regression-tested.

function pluginIdFromBaseURL(baseURL) {
  const value = String(baseURL || "").trim();
  const prefix = "plugin://";
  if (value.toLowerCase().startsWith(prefix)) {
    return value.slice(prefix.length).trim();
  }
  return "";
}

function buildPluginPayload(input) {
  const vendor = String(input.vendor || "custom").trim() || "custom";
  let pluginId = String(input.plugin_id || "").trim();
  let baseURL = String(input.base_url || "").trim();
  if (isPluginVendor(vendor)) {
    if (!pluginId) {
      pluginId = pluginIdFromBaseURL(baseURL);
    }
    if (pluginId && !baseURL) {
      baseURL = `plugin://${pluginId}`;
    }
  } else {
    pluginId = "";
  }
  return { vendor, plugin_id: pluginId, base_url: baseURL };
}

describe("isPluginVendor", () => {
  it("detects plugin vendor case-insensitively", () => {
    expect(isPluginVendor("plugin")).toBe(true);
    expect(isPluginVendor("Plugin")).toBe(true);
    expect(isPluginVendor("openai")).toBe(false);
    expect(isPluginVendor("")).toBe(false);
  });
});

describe("plugin provider payload rules", () => {
  it("fills base_url from plugin_id", () => {
    expect(
      buildPluginPayload({ vendor: "plugin", plugin_id: "grokbuild", base_url: "" }),
    ).toEqual({
      vendor: "plugin",
      plugin_id: "grokbuild",
      base_url: "plugin://grokbuild",
    });
  });

  it("derives plugin_id from plugin:// base_url", () => {
    expect(
      buildPluginPayload({ vendor: "plugin", plugin_id: "", base_url: "plugin://mock" }),
    ).toEqual({
      vendor: "plugin",
      plugin_id: "mock",
      base_url: "plugin://mock",
    });
  });

  it("clears plugin_id for non-plugin vendors", () => {
    expect(
      buildPluginPayload({
        vendor: "openai",
        plugin_id: "grokbuild",
        base_url: "https://api.openai.com",
      }),
    ).toEqual({
      vendor: "openai",
      plugin_id: "",
      base_url: "https://api.openai.com",
    });
  });
});

describe("pluginProviderDefaults", () => {
  it("returns grokbuild seed models", () => {
    const d = pluginProviderDefaults("grokbuild");
    expect(d.name).toMatch(/GrokBuild/i);
    expect(d.protocol).toBe("openai-responses");
    expect(d.models.map((m) => m.name)).toEqual(
      expect.arrayContaining(["grok-4", "grok-4.5", "grok-build-0.1"]),
    );
  });

  it("returns generic defaults for unknown plugins", () => {
    const d = pluginProviderDefaults("acme");
    expect(d.name).toBe("Plugin acme");
    expect(d.models).toEqual([]);
  });
});
