import { defineStore } from "pinia";
import { ref, watch } from "vue";

// 受控的颜色主题，保持桌面工具风格而不过度花哨
export const colorThemes = {
  blue: {
    name: "Windows Blue",
    color: "#0a64d8",
    hover: "#005fcb",
    pressed: "#0056b7",
    soft: "#e8f1fe",
  },
  slate: {
    name: "Slate",
    color: "#3d6fb4",
    hover: "#335f9b",
    pressed: "#284f83",
    soft: "#e9eff8",
  },
  forest: {
    name: "Forest",
    color: "#1f7a57",
    hover: "#176448",
    pressed: "#11513a",
    soft: "#e8f3ee",
  },
};

export const useThemeStore = defineStore("theme", () => {
  // 明暗主题
  const theme = ref(localStorage.getItem("theme") || "light");
  // 颜色主题
  const colorTheme = ref(localStorage.getItem("colorTheme") || "blue");

  const setTheme = (newTheme) => {
    theme.value = newTheme;
    localStorage.setItem("theme", newTheme);
    applyTheme(newTheme);
  };

  const toggleTheme = () => {
    const newTheme = theme.value === "dark" ? "light" : "dark";
    setTheme(newTheme);
  };

  const setColorTheme = (newColorTheme) => {
    colorTheme.value = newColorTheme;
    localStorage.setItem("colorTheme", newColorTheme);
    applyColorTheme(newColorTheme);
  };

  const applyTheme = (themeName) => {
    // Use class-based dark mode for shadcn compatibility
    if (themeName === "dark") {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
    // Keep data-theme for backward compatibility
    document.documentElement.setAttribute("data-theme", themeName);
  };

  const applyColorTheme = (colorThemeName) => {
    const colors = colorThemes[colorThemeName];
    if (colors) {
      document.documentElement.style.setProperty("--color-accent", colors.color);
      document.documentElement.style.setProperty("--color-accent-hover", colors.hover);
      document.documentElement.style.setProperty("--color-accent-pressed", colors.pressed);
      document.documentElement.style.setProperty("--color-accent-light", colors.soft);
      document.documentElement.style.setProperty("--color-accent-soft", colors.soft);
      document.documentElement.style.setProperty("--ui-accent", colors.color);
      document.documentElement.style.setProperty("--ui-accent-hover", colors.hover);
      document.documentElement.style.setProperty("--ui-accent-pressed", colors.pressed);
      document.documentElement.style.setProperty("--ui-accent-soft", colors.soft);
    }
  };

  const initTheme = () => {
    applyTheme(theme.value);
    applyColorTheme(colorTheme.value);
  };

  // 获取当前颜色主题信息
  const getCurrentColorTheme = () => {
    return colorThemes[colorTheme.value] || colorThemes.blue;
  };

  // 获取所有颜色主题列表
  const getColorThemeList = () => {
    return Object.entries(colorThemes).map(([key, value]) => ({
      key,
      ...value,
    }));
  };

  return {
    theme,
    colorTheme,
    setTheme,
    toggleTheme,
    setColorTheme,
    initTheme,
    getCurrentColorTheme,
    getColorThemeList,
    colorThemes,
  };
});
