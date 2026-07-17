import { defineStore } from "pinia";
import {
  CheckSupplier,
  DeleteSupplier,
  FetchModelsFromProvider,
  GetSuppliersPage,
  ListRoutingRules,
  ListSupplierHealth,
  ListSuppliers,
  SaveRoutePolicy,
  SaveSupplier,
} from "../lib/apiClient";
import { DEFAULT_PAGE_SIZE } from "../constants/index";

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

const buildSupplierPayload = (form) => {
  const models = normalizeModels(form.models);
  return {
    ...form,
    models,
  };
};

const emptyForm = () => ({
  id: "",
  name: "",
  protocol: "openai-responses",
  vendor: "openai",
  plugin_id: "",
  base_url: "",
  models_url: "",
  proxy_url: "",
  api_key: "",
  only_stream: false,
  user_agent: "",
  enabled: true,
  description: "",
  models: [createEmptyModelItem()],
});

const isPluginVendorForm = (vendor) =>
  String(vendor || "").trim().toLowerCase() === "plugin";

const buildProviderPayload = (form) => {
  const vendor = form.vendor || "openai";
  const pluginId = isPluginVendorForm(vendor)
    ? String(form.plugin_id || "").trim()
    : "";
  let baseURL = String(form.base_url || "").trim();
  if (isPluginVendorForm(vendor) && pluginId && !baseURL) {
    baseURL = `plugin://${pluginId}`;
  }
  return {
    id: form.id,
    name: form.name,
    protocol: form.protocol,
    vendor,
    plugin_id: pluginId,
    base_url: baseURL,
    models_url: form.models_url,
    proxy_url: form.proxy_url,
    api_key: form.api_key,
    only_stream: form.only_stream,
    user_agent: form.user_agent,
    enabled: form.enabled,
    description: form.description,
    models: null,
  };
};

const emptyPolicyForm = () => ({
  id: "",
  downstream_protocol: "anthropic",
  upstream_protocol: "",
  supplier_id: "",
  enabled: true,
});

const emptyModelForm = () => ({
  ...emptyForm(),
});

const isCatchAllPattern = (pattern) => {
  const value = String(pattern || "").trim();
  return value === "" || value === "*";
};

const compareRoutingRules = (left, right) => {
  const priorityDiff = Number(left?.priority || 0) - Number(right?.priority || 0);
  if (priorityDiff !== 0) {
    return priorityDiff;
  }
  return String(left?.name || "").localeCompare(String(right?.name || ""));
};

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
    routingRules: [],
    health: [],
    total: 0,
    totalCount: 0,
    enabledCount: 0,
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
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
        // 同一协议下若存在多条默认规则，保留第一条（按优先级/名称已排序），避免后覆盖前
        if (!lookup[item.downstream_protocol]) {
          lookup[item.downstream_protocol] = item;
        }
      });
      return this.routeDefinitions.map((definition) => ({
        ...definition,
        policy: lookup[definition.key] || null,
      }));
    },
    routeManagementRows() {
      const providerLookup = Object.fromEntries(
        this.allSuppliers.map((item) => [item.id, item]),
      );
      const enabledRules = [...this.routingRules]
        .filter((item) => item.enabled)
        .sort(compareRoutingRules);

      return this.policiesByProtocol.map((item) => {
        if (!item.policy) {
          return {
            ...item,
            supplierName: "未分配",
            upstreamProtocol: "待选择",
            warningText: "",
            statusText: "未配置",
            statusVariant: "warning",
          };
        }

        const higherPriorityRules = enabledRules.filter((rule) =>
          rule.downstream_protocol === item.key &&
          rule.id !== item.policy.id &&
          Number(rule.priority || 0) < Number(item.policy.priority || 0),
        );
        const blockingRule = higherPriorityRules.find((rule) => isCatchAllPattern(rule.match_model_pattern));
        const partialRules = higherPriorityRules.filter((rule) => !isCatchAllPattern(rule.match_model_pattern));

        let warningText = "";
        if (blockingRule) {
          const providerName = providerLookup[blockingRule.target_provider_id]?.name || blockingRule.supplier_name || "未命名供应商";
          warningText = `默认路由当前不会生效。更高优先级规则“${blockingRule.name}”会优先转到 ${providerName}。`;
        } else if (partialRules.length > 0) {
          const names = partialRules.slice(0, 2).map((rule) => `“${rule.name}”`).join("、");
          const suffix = partialRules.length > 2 ? ` 等 ${partialRules.length} 条规则` : "";
          warningText = `存在更高优先级模型规则 ${names}${suffix}，命中这些模型时不会走默认路由。`;
        }

        return {
          ...item,
          supplierName: item.policy.supplier_name || "未分配",
          upstreamProtocol: item.policy.upstream_protocol || "待选择",
          warningText,
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
      const [allSuppliers, rules] = await Promise.all([
        ListSuppliers(),
        ListRoutingRules(),
      ]);
      this.allSuppliers = allSuppliers;
      this.routingRules = rules;
      this.policies = rules.filter((item) => isCatchAllPattern(item.match_model_pattern));
    },
    async load() {
      this.loading = true;
      this.error = "";
      try {
        const [pageResult, allSuppliers, rules, health] = await Promise.all([
          GetSuppliersPage(1, this.pageSize, this.keyword, this.protocol),
          ListSuppliers(),
          ListRoutingRules(),
          ListSupplierHealth(),
        ]);
        this.applySupplierPage(pageResult, 1, this.pageSize);
        this.allSuppliers = allSuppliers;
        this.routingRules = rules;
        this.policies = rules.filter((item) => isCatchAllPattern(item.match_model_pattern));
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
      const vendor = item.vendor || "openai";
      const pluginId =
        item.plugin_id ||
        (isPluginVendorForm(vendor)
          ? String(item.base_url || "").replace(/^plugin:\/\//i, "").trim()
          : "");
      this.form = {
        id: item.id,
        name: item.name,
        protocol: item.protocol,
        vendor,
        plugin_id: pluginId,
        base_url: item.base_url,
        models_url: item.models_url || "",
        proxy_url: item.proxy_url || "",
        api_key: "",
        only_stream: Boolean(item.only_stream),
        user_agent: item.user_agent || "",
        enabled: Boolean(item.enabled),
        description: item.description || "",
        models: cloneModelsForForm(item.models),
      };
    },
    resetForm() {
      this.form = emptyForm();
    },
    selectModelEditor(item) {
      const vendor = item.vendor || "openai";
      const pluginId =
        item.plugin_id ||
        (isPluginVendorForm(vendor)
          ? String(item.base_url || "").replace(/^plugin:\/\//i, "").trim()
          : "");
      this.modelForm = {
        id: item.id,
        name: item.name,
        protocol: item.protocol,
        vendor,
        plugin_id: pluginId,
        base_url: item.base_url,
        models_url: item.models_url || "",
        proxy_url: item.proxy_url || "",
        api_key: "",
        only_stream: Boolean(item.only_stream),
        user_agent: item.user_agent || "",
        enabled: Boolean(item.enabled),
        description: item.description || "",
        models: cloneModelsForForm(item.models),
      };
    },
    resetModelForm() {
      this.modelForm = emptyModelForm();
    },
    selectPolicy(item) {
      this.policyForm = {
        id: item.id,
        downstream_protocol: item.downstream_protocol,
        upstream_protocol: item.upstream_protocol || "",
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
        await SaveSupplier(buildProviderPayload(this.form));
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
    async savePolicy({ force = false, silentActiveRuleError = false } = {}) {
      this.saving = true;
      this.error = "";
      try {
        await SaveRoutePolicy({ ...this.policyForm, force });
        await this.refreshSupplierCatalog();
        this.resetPolicyForm();
        return { ok: true, error: "" };
      } catch (error) {
        const message = error?.message || String(error);
        if (!(silentActiveRuleError && message.includes("active requests"))) {
          this.error = message;
        }
        return { ok: false, error: message };
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
        const record = await CheckSupplier(id);
        this.health = [
          ...this.health.filter((item) => item.supplier_id !== id),
          record,
        ];
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.checking = "";
      }
    },
    async fetchModels(providerID) {
      this.fetchingModels = true;
      this.error = "";
      try {
        const fetched = await FetchModelsFromProvider(providerID);
        if (!fetched?.length) {
          return 0;
        }
        // Merge fetched models into the form's model list.
        const existingNames = new Set(
          (this.modelForm.models || []).map((m) => String(m?.name || "").trim()),
        );
        const newModels = fetched
          .filter((m) => !m.exists && !existingNames.has(m.name))
          .map((m) => ({
            name: m.name,
            max_tokens: m.max_tokens || 32768,
          }));
        if (newModels.length) {
          this.modelForm.models = [...(this.modelForm.models || []), ...newModels];
        }
        return newModels.length;
      } catch (error) {
        this.error = error?.message || String(error);
        return 0;
      } finally {
        this.fetchingModels = false;
      }
    },
  },
});
