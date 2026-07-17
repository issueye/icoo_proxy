import { defineStore } from "pinia";
import { GetUiPrefs, SaveUiPrefs } from "../lib/apiClient";

const defaultPrefs = () => ({
  theme: "blue",
  /** Default control height preset — tightened with global UED compact baseline. */
  buttonSize: "sm",
  /** Layout density: compact (紧缩) | comfortable (宽松) */
  density: "compact",
});

const themes = {
  blue: {
    "--ued-color-primary": "#1677ff",
    "--ued-color-primary-hover": "#4096ff",
    "--ued-color-primary-active": "#0958d9",
    "--ued-color-primary-foreground": "#ffffff",
    "--ued-color-primary-soft": "#e6f4ff",
  },
  green: {
    "--ued-color-primary": "#52c41a",
    "--ued-color-primary-hover": "#73d13d",
    "--ued-color-primary-active": "#389e0d",
    "--ued-color-primary-foreground": "#ffffff",
    "--ued-color-primary-soft": "#f6ffed",
  },
  purple: {
    "--ued-color-primary": "#722ed1",
    "--ued-color-primary-hover": "#9254de",
    "--ued-color-primary-active": "#531dab",
    "--ued-color-primary-foreground": "#ffffff",
    "--ued-color-primary-soft": "#f9f0ff",
  },
  orange: {
    "--ued-color-primary": "#fa8c16",
    "--ued-color-primary-hover": "#ffa940",
    "--ued-color-primary-active": "#d46b08",
    "--ued-color-primary-foreground": "#ffffff",
    "--ued-color-primary-soft": "#fff7e6",
  },
  red: {
    "--ued-color-primary": "#f5222d",
    "--ued-color-primary-hover": "#ff4d4f",
    "--ued-color-primary-active": "#cf1322",
    "--ued-color-primary-foreground": "#ffffff",
    "--ued-color-primary-soft": "#fff1f0",
  },
  cyan: {
    "--ued-color-primary": "#13c2c2",
    "--ued-color-primary-hover": "#36cfc9",
    "--ued-color-primary-active": "#08979c",
    "--ued-color-primary-foreground": "#ffffff",
    "--ued-color-primary-soft": "#e6fffb",
  },
  dark: {
    "--ued-color-primary": "#141414",
    "--ued-color-primary-hover": "#262626",
    "--ued-color-primary-active": "#000000",
    "--ued-color-primary-foreground": "#ffffff",
    "--ued-color-primary-soft": "#f5f5f5",
  },
};

const buttonSizes = {
  xs: {
    "--ued-size-xs": "20px",
    "--ued-size-sm": "22px",
    "--ued-size-md": "24px",
    "--ued-size-lg": "30px",
  },
  sm: {
    "--ued-size-xs": "21px",
    "--ued-size-sm": "24px",
    "--ued-size-md": "26px",
    "--ued-size-lg": "32px",
  },
  md: {
    "--ued-size-xs": "22px",
    "--ued-size-sm": "26px",
    "--ued-size-md": "28px",
    "--ued-size-lg": "34px",
  },
  lg: {
    "--ued-size-xs": "24px",
    "--ued-size-sm": "28px",
    "--ued-size-md": "32px",
    "--ued-size-lg": "40px",
  },
};

/**
 * Density modes scale page spacing, panel gaps, and table row heights.
 * Control heights are still refined by buttonSize after density is applied.
 *
 * compact     — 紧缩：高信息密度（控制台默认）
 * comfortable — 宽松：更大留白与行高，阅读更轻松
 */
const densityModes = {
  /** 紧缩 — 全局 UED 收紧后的默认运维台密度 */
  compact: {
    "--ued-space-1": "2px",
    "--ued-space-2": "3px",
    "--ued-space-3": "4px",
    "--ued-space-4": "6px",
    "--ued-space-5": "8px",
    "--ued-space-6": "10px",
    "--ued-space-7": "10px",
    "--ued-space-8": "12px",
    "--ued-space-10": "14px",
    "--ued-space-12": "16px",
    "--ued-space-16": "24px",
    "--ued-space-page": "8px",
    "--ued-space-section": "8px",
    "--ued-space-panel": "8px",
    "--ued-space-panel-sm": "6px",
    "--ued-space-stack": "6px",
    "--ued-space-inline": "4px",
    "--ued-space-control": "6px",
    "--ued-space-table-x": "6px",
    "--ued-space-table-cell-x": "8px",
    "--ued-table-header-height": "28px",
    "--ued-table-row-height": "30px",
    "--ued-font-size-base": "12px",
    "--ued-font-size-sm": "11px",
    "--ued-font-size-xs": "10px",
    "--ued-size-xs": "20px",
    "--ued-size-sm": "22px",
    "--ued-size-md": "24px",
    "--ued-size-lg": "28px",
  },
  /** 宽松 — 仍比旧版舒适，相对紧缩有明显呼吸感 */
  comfortable: {
    "--ued-space-1": "2px",
    "--ued-space-2": "4px",
    "--ued-space-3": "6px",
    "--ued-space-4": "8px",
    "--ued-space-5": "10px",
    "--ued-space-6": "12px",
    "--ued-space-7": "12px",
    "--ued-space-8": "14px",
    "--ued-space-10": "16px",
    "--ued-space-12": "20px",
    "--ued-space-16": "28px",
    "--ued-space-page": "12px",
    "--ued-space-section": "12px",
    "--ued-space-panel": "12px",
    "--ued-space-panel-sm": "8px",
    "--ued-space-stack": "8px",
    "--ued-space-inline": "6px",
    "--ued-space-control": "8px",
    "--ued-space-table-x": "8px",
    "--ued-space-table-cell-x": "12px",
    "--ued-table-header-height": "32px",
    "--ued-table-row-height": "36px",
    "--ued-font-size-base": "13px",
    "--ued-font-size-sm": "12px",
    "--ued-font-size-xs": "11px",
    "--ued-size-xs": "22px",
    "--ued-size-sm": "26px",
    "--ued-size-md": "28px",
    "--ued-size-lg": "34px",
  },
};

function applyVars(root, vars) {
  for (const [key, value] of Object.entries(vars)) {
    root.style.setProperty(key, value);
  }
}

function applyToDocument(prefs) {
  const root = document.documentElement;
  const themeVars = themes[prefs.theme] || themes.blue;
  applyVars(root, themeVars);

  const density = densityModes[prefs.density] ? prefs.density : "compact";
  applyVars(root, densityModes[density]);

  // buttonSize fine-tunes control heights on top of density base.
  const sizeVars = buttonSizes[prefs.buttonSize] || buttonSizes.md;
  applyVars(root, sizeVars);

  root.setAttribute("data-theme", prefs.theme);
  root.setAttribute("data-button-size", prefs.buttonSize);
  root.setAttribute("data-density", density);
}

function persistPayload(prefs) {
  return {
    theme: prefs.theme,
    buttonSize: prefs.buttonSize,
    density: prefs.density,
  };
}

export const useUiPrefsStore = defineStore("uiPrefs", {
  state: () => ({
    prefs: defaultPrefs(),
    ready: false,
  }),
  getters: {
    theme: (state) => state.prefs.theme,
    buttonSize: (state) => state.prefs.buttonSize,
    density: (state) => state.prefs.density,
    themeOptions: () => [
      { label: "蓝色", value: "blue", color: "#1677ff" },
      { label: "绿色", value: "green", color: "#52c41a" },
      { label: "紫色", value: "purple", color: "#722ed1" },
      { label: "橙色", value: "orange", color: "#fa8c16" },
      { label: "红色", value: "red", color: "#f5222d" },
      { label: "青色", value: "cyan", color: "#13c2c2" },
      { label: "深色", value: "dark", color: "#141414" },
    ],
    buttonSizeOptions: () => [
      { label: "紧凑 (XS)", value: "xs" },
      { label: "偏小 (SM)", value: "sm" },
      { label: "默认 (MD)", value: "md" },
      { label: "偏大 (LG)", value: "lg" },
    ],
    densityOptions: () => [
      {
        label: "紧缩",
        value: "compact",
        description: "高信息密度，适合运维台与宽表",
      },
      {
        label: "宽松",
        value: "comfortable",
        description: "更大留白与行高，阅读更轻松",
      },
    ],
  },
  actions: {
    init() {
      applyToDocument(this.prefs);
      this.loadFromBackend();
    },
    async loadFromBackend() {
      try {
        const prefs = await GetUiPrefs();
        if (prefs && prefs.theme) {
          this.prefs.theme = prefs.theme;
          this.prefs.buttonSize = prefs.buttonSize || "sm";
          this.prefs.density =
            prefs.density === "comfortable" || prefs.density === "compact"
              ? prefs.density
              : "compact";
          applyToDocument(this.prefs);
        }
      } catch {
        // use defaults already applied
      } finally {
        this.ready = true;
      }
    },
    async setTheme(theme) {
      this.prefs.theme = theme;
      applyToDocument(this.prefs);
      try {
        await SaveUiPrefs(persistPayload(this.prefs));
      } catch {
        // saved locally even if backend fails
      }
    },
    async setButtonSize(size) {
      this.prefs.buttonSize = size;
      applyToDocument(this.prefs);
      try {
        await SaveUiPrefs(persistPayload(this.prefs));
      } catch {
        // saved locally even if backend fails
      }
    },
    async setDensity(density) {
      if (density !== "compact" && density !== "comfortable") {
        density = "compact";
      }
      this.prefs.density = density;
      // Align control size preset with density for a coherent first switch.
      if (density === "comfortable" && (this.prefs.buttonSize === "sm" || this.prefs.buttonSize === "xs")) {
        this.prefs.buttonSize = "md";
      } else if (density === "compact" && (this.prefs.buttonSize === "md" || this.prefs.buttonSize === "lg")) {
        this.prefs.buttonSize = "sm";
      }
      applyToDocument(this.prefs);
      try {
        await SaveUiPrefs(persistPayload(this.prefs));
      } catch {
        // saved locally even if backend fails
      }
    },
  },
});
