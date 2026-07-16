import { defineStore } from "pinia";
import { GetProjectSettings, SaveProjectSettings } from "../lib/apiClient";

const emptyForm = () => ({
  proxy_host: "127.0.0.1",
  proxy_port: 18181,
  proxy_read_timeout_seconds: 15,
  proxy_write_timeout_seconds: 300,
  proxy_shutdown_timeout_seconds: 10,
  default_max_tokens: 32768,
  proxy_chain_log_path: ".data/bridge-chain.log",
  proxy_chain_log_bodies: false,
  proxy_chain_log_max_body_bytes: 8192,
});

export const useSettingsStore = defineStore("settings", {
  state: () => ({
    loading: false,
    saving: false,
    error: "",
    success: "",
    form: emptyForm(),
  }),
  actions: {
    async load() {
      this.loading = true;
      this.error = "";
      this.success = "";
      try {
        const settings = await GetProjectSettings();
        this.form = { ...emptyForm(), ...settings };
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.loading = false;
      }
    },
    async save() {
      this.saving = true;
      this.error = "";
      this.success = "";
      try {
        const payload = {
          ...this.form,
          proxy_port: Number(this.form.proxy_port),
          proxy_read_timeout_seconds: Number(this.form.proxy_read_timeout_seconds),
          proxy_write_timeout_seconds: Number(this.form.proxy_write_timeout_seconds),
          proxy_shutdown_timeout_seconds: Number(this.form.proxy_shutdown_timeout_seconds),
          default_max_tokens: Number(this.form.default_max_tokens),
          proxy_chain_log_max_body_bytes: Number(this.form.proxy_chain_log_max_body_bytes),
        };
        await SaveProjectSettings(payload);
        this.success = "设置已保存。";
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.saving = false;
      }
    },
    reset() {
      this.form = emptyForm();
      this.error = "";
      this.success = "";
    },
  },
});
