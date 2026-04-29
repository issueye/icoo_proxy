import { defineStore } from "pinia";
import { ClearTrafficRequests, GetTrafficPage } from "../lib/wailsApp";

function defaultTokenStats() {
  return {
    input_tokens: 0,
    output_tokens: 0,
    total_tokens: 0,
  };
}

export const useTrafficStore = defineStore("traffic", {
  state: () => ({
    loading: false,
    refreshing: false,
    clearing: false,
    error: "",
    requests: [],
    tokenStats: defaultTokenStats(),
    totalRequests: 0,
    total: 0,
    page: 1,
    pageSize: 8,
    successCount: 0,
    errorCount: 0,
    averageLatency: 0,
    protocolOptions: ["all"],
    filter: "all",
    autoRefresh: false,
    lastUpdatedAt: "",
  }),
  getters: {
    hasData(state) {
      return state.total > 0;
    },
  },
  actions: {
    async fetchPage({ page = this.page, pageSize = this.pageSize, refreshing = false } = {}) {
      this.loading = !refreshing;
      this.refreshing = refreshing;
      this.error = "";

      try {
        const result = await GetTrafficPage(page, pageSize, this.filter);

        this.requests = result?.items || [];
        this.total = Number(result?.total || 0);
        this.page = Number(result?.page || page || 1);
        this.pageSize = Number(result?.page_size || pageSize || 8);
        this.protocolOptions = result?.protocol_options || ["all"];
        this.tokenStats = result?.token_stats || defaultTokenStats();
        this.totalRequests = Number(result?.total_requests || 0);
        this.successCount = Number(result?.success_count || 0);
        this.errorCount = Number(result?.error_count || 0);
        this.averageLatency = Number(result?.average_latency || 0);
        this.lastUpdatedAt = result?.last_updated_at || new Date().toISOString();

        if (this.total > 0 && this.requests.length === 0 && this.page > 1) {
          const maxPage = Math.max(1, Math.ceil(this.total / this.pageSize));
          if (maxPage !== this.page) {
            this.page = maxPage;
            return await this.fetchPage({ page: this.page, pageSize: this.pageSize, refreshing });
          }
        }
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.loading = false;
        this.refreshing = false;
      }
    },
    async load() {
      await this.fetchPage({ page: 1, pageSize: this.pageSize });
    },
    async refresh() {
      await this.fetchPage({ refreshing: true });
    },
    async clear() {
      this.clearing = true;
      this.error = "";
      try {
        await ClearTrafficRequests();
        this.requests = [];
        this.tokenStats = defaultTokenStats();
        this.totalRequests = 0;
        this.total = 0;
        this.successCount = 0;
        this.errorCount = 0;
        this.averageLatency = 0;
        this.page = 1;
        this.protocolOptions = ["all"];
        this.filter = "all";
        this.lastUpdatedAt = new Date().toISOString();
        await this.fetchPage({ page: 1, pageSize: this.pageSize, refreshing: true });
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.clearing = false;
      }
    },
    async setFilter(value) {
      this.filter = value;
      this.page = 1;
      await this.fetchPage({ page: 1, pageSize: this.pageSize });
    },
    async changePage({ page, pageSize }) {
      await this.fetchPage({
        page: page || this.page,
        pageSize: pageSize || this.pageSize,
      });
    },
    toggleAutoRefresh() {
      this.autoRefresh = !this.autoRefresh;
    },
  },
});
