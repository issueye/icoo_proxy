import { defineStore } from "pinia";
import {
  CheckSupplier,
  DeleteSupplier,
  GetSuppliersPage,
  ListRoutePolicies,
  ListSupplierHealth,
  ListSuppliers,
  SaveRoutePolicy,
  SaveSupplier,
} from "../lib/wailsApp";

const routeDefinitions = [
  {
    key: "anthropic",
    label: "Anthropic",
    description: "",
  },
  {
    key: "openai-chat",
    label: "OpenAI Chat",
    description: "",
  },
  {
    key: "openai-responses",
    label: "OpenAI Responses",
    description: "",
  },
];

const protocolOptions = routeDefinitions.map((item) => ({
  label: item.label,
  value: item.key,
}));

const DEFAULT_MODEL_MAX_TOKENS = 32768;

const createEmptyModelItem = () => ({
  name: "",
  max_tokens: DEFAULT_MODEL_MAX_TOKENS,
});

const normalizeMaxTokens = (value) => {
  const parsed = Number.parseInt(value, 10);
  return parsed > 0 ? parsed : DEFAULT_MODEL_MAX_TOKENS;
};

const normalizeModelItem = (item) => {
  if (typeof item === "string") {
    const name = item.trim();
    return name
      ? {
          name,
          max_tokens: DEFAULT_MODEL_MAX_TOKENS,
        }
      : null;
  }

  const name = String(item?.name || "").trim();
  if (!name) {
    return null;
  }

  return {
    name,
    max_tokens: normalizeMaxTokens(item?.max_tokens),
  };
};

const normalizeModels = (models) =>
  (models || [])
    .map(normalizeModelItem)
    .filter(Boolean);

const cloneModelsForForm = (models) => {
  const normalized = normalizeModels(models);
  return normalized.length ? normalized : [createEmptyModelItem()];
};

const normalizeDefaultModel = (models, defaultModel) => {
  const target = String(defaultModel || "").trim();
  if (!target) {
    return "";
  }

  const matched = normalizeModels(models).find((item) => item.name === target);
  return matched?.name || "";
};

const buildSupplierPayload = (form) => {
  const models = normalizeModels(form.models);
  return {
    ...form,
    default_model: normalizeDefaultModel(models, form.default_model),
    models,
  };
};

const emptyForm = () => ({
  id: "",
  name: "",
  protocol: "openai-responses",
  vendor: "openai",
  base_url: "",
  api_key: "",
  only_stream: false,
  user_agent: "",
  enabled: true,
  description: "",
  models: [createEmptyModelItem()],
  default_model: "",
});

const emptyPolicyForm = () => ({
  id: "",
  downstream_protocol: "anthropic",
  supplier_id: "",
  enabled: true,
});

const emptyModelForm = () => ({
  ...emptyForm(),
});

export const useSuppliersStore = defineStore("suppliers", {
  state: () => ({
    loading: false,
    saving: false,
    deleting: "",
    checking: "",
    error: "",
    items: [],
    allSuppliers: [],
    policies: [],
    health: [],
    total: 0,
    totalCount: 0,
    enabledCount: 0,
    page: 1,
    pageSize: 8,
    keyword: "",
    protocol: "all",
    form: emptyForm(),
    modelForm: emptyModelForm(),
    policyForm: emptyPolicyForm(),
  }),
  getters: {
    checkedCount(state) {
      return state.health.length;
    },
    configuredPolicyCount(state) {
      return state.policies.filter((item) => item.supplier_id).length;
    },
    enabledPolicyCount(state) {
      return state.policies.filter((item) => item.enabled).length;
    },
    routeDefinitions() {
      return routeDefinitions;
    },
    policyOptions() {
      return protocolOptions;
    },
    policiesByProtocol() {
      const lookup = {};
      this.policies.forEach((item) => {
        lookup[item.downstream_protocol] = item;
      });
      return this.routeDefinitions.map((definition) => ({
        ...definition,
        policy: lookup[definition.key] || null,
      }));
    },
    routeManagementRows() {
      const supplierLookup = {};
      this.allSuppliers.forEach((item) => {
        supplierLookup[item.id] = item;
      });
      return this.policiesByProtocol.map((item) => {
        if (!item.policy) {
          return {
            ...item,
            supplierName: "未分配",
            upstreamProtocol: "待选择",
            helperText: "默认模型将继承所选供应商配置。",
            statusText: "未配置",
            statusVariant: "warning",
          };
        }
        const supplier = supplierLookup[item.policy.supplier_id] || null;
        return {
          ...item,
          supplierName: item.policy.supplier_name || "未分配",
          upstreamProtocol: item.policy.upstream_protocol || "待选择",
          helperText: supplier?.default_model
            ? `默认模型：${supplier.default_model}`
            : "该供应商尚未配置默认模型。",
          statusText: item.policy.enabled ? "已启用" : "已停用",
          statusVariant: item.policy.enabled ? "success" : "error",
        };
      });
    },
  },
  actions: {
    applySupplierPage(result, fallbackPage = this.page, fallbackPageSize = this.pageSize) {
      this.items = result?.items || [];
      this.total = Number(result?.total || 0);
      this.totalCount = Number(result?.total_count || 0);
      this.enabledCount = Number(result?.enabled_count || 0);
      this.page = Number(result?.page || fallbackPage || 1);
      this.pageSize = Number(result?.page_size || fallbackPageSize || 8);
    },
    async fetchPage({ page = this.page, pageSize = this.pageSize } = {}) {
      const result = await GetSuppliersPage(page, pageSize, this.keyword, this.protocol);
      this.applySupplierPage(result, page, pageSize);
    },
    async refreshSupplierCatalog() {
      const [allSuppliers, policies] = await Promise.all([
        ListSuppliers(),
        ListRoutePolicies(),
      ]);
      this.allSuppliers = allSuppliers;
      this.policies = policies;
    },
    async load() {
      this.loading = true;
      this.error = "";
      try {
        const [pageResult, allSuppliers, policies, health] = await Promise.all([
          GetSuppliersPage(1, this.pageSize, this.keyword, this.protocol),
          ListSuppliers(),
          ListRoutePolicies(),
          ListSupplierHealth(),
        ]);
        this.applySupplierPage(pageResult, 1, this.pageSize);
        this.allSuppliers = allSuppliers;
        this.policies = policies;
        this.health = health;
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.loading = false;
      }
    },
    async changePage({ page, pageSize }) {
      this.loading = true;
      this.error = "";
      try {
        await this.fetchPage({
          page: page || this.page,
          pageSize: pageSize || this.pageSize,
        });
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.loading = false;
      }
    },
    async applyFilters(filters = {}) {
      this.loading = true;
      this.error = "";
      try {
        this.keyword = String(filters.keyword ?? this.keyword ?? "").trim();
        this.protocol = this.policyOptions.some((item) => item.value === filters.protocol)
          ? filters.protocol
          : "all";
        this.page = 1;
        await this.fetchPage({ page: 1, pageSize: this.pageSize });
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.loading = false;
      }
    },
    async resetFilters() {
      await this.applyFilters({
        keyword: "",
        protocol: "all",
      });
    },
    select(item) {
      this.form = {
        id: item.id,
        name: item.name,
        protocol: item.protocol,
        vendor: item.vendor || "openai",
        base_url: item.base_url,
        api_key: "",
        only_stream: Boolean(item.only_stream),
        user_agent: item.user_agent || "",
        enabled: Boolean(item.enabled),
        description: item.description || "",
        models: cloneModelsForForm(item.models),
        default_model: item.default_model || "",
      };
    },
    resetForm() {
      this.form = emptyForm();
    },
    selectModelEditor(item) {
      this.modelForm = {
        id: item.id,
        name: item.name,
        protocol: item.protocol,
        vendor: item.vendor || "openai",
        base_url: item.base_url,
        api_key: "",
        only_stream: Boolean(item.only_stream),
        user_agent: item.user_agent || "",
        enabled: Boolean(item.enabled),
        description: item.description || "",
        models: cloneModelsForForm(item.models),
        default_model: item.default_model || "",
      };
    },
    resetModelForm() {
      this.modelForm = emptyModelForm();
    },
    selectPolicy(item) {
      this.policyForm = {
        id: item.id,
        downstream_protocol: item.downstream_protocol,
        supplier_id: item.supplier_id,
        enabled: Boolean(item.enabled),
      };
    },
    resetPolicyForm() {
      this.policyForm = emptyPolicyForm();
    },
    healthFor(id) {
      return this.health.find((item) => item.supplier_id === id);
    },
    async save() {
      this.saving = true;
      this.error = "";
      try {
        await SaveSupplier(buildSupplierPayload(this.form));
        await Promise.all([
          this.fetchPage({ page: this.page, pageSize: this.pageSize }),
          this.refreshSupplierCatalog(),
        ]);
        this.resetForm();
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.saving = false;
      }
    },
    async saveModelEditor() {
      this.saving = true;
      this.error = "";
      try {
        await SaveSupplier(buildSupplierPayload(this.modelForm));
        await Promise.all([
          this.fetchPage({ page: this.page, pageSize: this.pageSize }),
          this.refreshSupplierCatalog(),
        ]);
        if (this.form.id) {
          const current = this.allSuppliers.find((item) => item.id === this.form.id);
          if (current) {
            this.select(current);
          }
        }
        this.resetModelForm();
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.saving = false;
      }
    },
    async savePolicy() {
      this.saving = true;
      this.error = "";
      try {
        this.policies = await SaveRoutePolicy({ ...this.policyForm });
        this.resetPolicyForm();
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
        await DeleteSupplier(id);
        await Promise.all([
          this.fetchPage({ page: this.page, pageSize: this.pageSize }),
          this.refreshSupplierCatalog(),
        ]);
        this.health = this.health.filter((item) => item.supplier_id !== id);
        if (this.form.id === id) {
          this.resetForm();
        }
        if (this.modelForm.id === id) {
          this.resetModelForm();
        }
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.deleting = "";
      }
    },
    async check(id) {
      this.checking = id;
      this.error = "";
      try {
        this.health = await CheckSupplier(id);
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.checking = "";
      }
    },
  },
});
