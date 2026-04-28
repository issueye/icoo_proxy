import { defineStore } from "pinia";
import { DeleteEndpoint, GetEndpointsPage, ReloadProxy, SaveEndpoint } from "../lib/wailsApp";

const emptyForm = () => ({
  id: "",
  path: "",
  protocol: "openai-chat",
  description: "",
  enabled: true,
});

export const useEndpointsStore = defineStore("endpoints", {
  state: () => ({
    loading: false,
    saving: false,
    deleting: "",
    reloading: false,
    error: "",
    items: [],
    total: 0,
    totalCount: 0,
    enabledCount: 0,
    customCount: 0,
    page: 1,
    pageSize: 8,
    keyword: "",
    protocol: "all",
    form: emptyForm(),
  }),
  getters: {
    protocolOptions() {
      return [
        { label: "anthropic", value: "anthropic" },
        { label: "openai-chat", value: "openai-chat" },
        { label: "openai-responses", value: "openai-responses" },
      ];
    },
    filterProtocolOptions() {
      return [
        { label: "全部协议", value: "all" },
        ...this.protocolOptions,
      ];
    },
  },
  actions: {
    async fetchPage({ page = this.page, pageSize = this.pageSize } = {}) {
      this.loading = true;
      this.error = "";
      try {
        const result = await GetEndpointsPage(page, pageSize, this.keyword, this.protocol);
        this.items = result?.items || [];
        this.total = Number(result?.total || 0);
        this.totalCount = Number(result?.total_count || 0);
        this.enabledCount = Number(result?.enabled_count || 0);
        this.customCount = Number(result?.custom_count || 0);
        this.page = Number(result?.page || page || 1);
        this.pageSize = Number(result?.page_size || pageSize || 8);
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.loading = false;
      }
    },
    async load() {
      await this.fetchPage({ page: 1, pageSize: this.pageSize });
    },
    async changePage({ page, pageSize }) {
      await this.fetchPage({
        page: page || this.page,
        pageSize: pageSize || this.pageSize,
      });
    },
    async applyFilters(filters = {}) {
      this.keyword = String(filters.keyword ?? this.keyword ?? "").trim();
      this.protocol = this.filterProtocolOptions.some((item) => item.value === filters.protocol)
        ? filters.protocol
        : "all";
      this.page = 1;
      await this.fetchPage({ page: 1, pageSize: this.pageSize });
    },
    async resetFilters() {
      this.keyword = "";
      this.protocol = "all";
      this.page = 1;
      await this.fetchPage({ page: 1, pageSize: this.pageSize });
    },
    select(item) {
      this.form = {
        id: item.id,
        path: item.path,
        protocol: item.protocol,
        description: item.description || "",
        enabled: Boolean(item.enabled),
      };
    },
    resetForm() {
      this.form = emptyForm();
    },
    async save() {
      this.saving = true;
      this.error = "";
      try {
        await SaveEndpoint({ ...this.form });
        this.resetForm();
        await this.fetchPage({ page: this.page, pageSize: this.pageSize });
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.saving = false;
      }
    },
    async remove(id) {
      this.deleting = id;
      this.error = "";
      try {
        await DeleteEndpoint(id);
        if (this.form.id === id) {
          this.resetForm();
        }
        await this.fetchPage({ page: this.page, pageSize: this.pageSize });
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.deleting = "";
      }
    },
    async reloadProxy() {
      this.reloading = true;
      this.error = "";
      try {
        await ReloadProxy();
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.reloading = false;
      }
    },
  },
});
