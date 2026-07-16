/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{vue,js}"],
  theme: {
    extend: {
      screens: {
        sm: "480px",
        md: "640px",
        lg: "800px",
        xl: "1024px",
      },
      colors: {
        // UI colors are driven by CSS custom properties in main.css.
        // Avoid adding duplicate palettes here; use token classes instead.
      },
      boxShadow: {
        panel: "0 1px 2px rgba(0, 0, 0, 0.03), 0 4px 12px rgba(0, 0, 0, 0.04)",
        popup:
          "0 6px 16px rgba(0, 0, 0, 0.08), 0 3px 6px -4px rgba(0, 0, 0, 0.12), 0 9px 28px 8px rgba(0, 0, 0, 0.05)",
      },
      fontFamily: {
        sans: ["Segoe UI", "PingFang SC", "Microsoft YaHei", "sans-serif"],
        mono: ["Cascadia Code", "Consolas", "monospace"],
      },
    },
  },
  plugins: [],
};
