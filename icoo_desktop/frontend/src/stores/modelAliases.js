import { defineStore } from "pinia";
import {
  DeleteModelAlias,
  FetchModelsFromProvider,
  ListModelAliases,
  ListSuppliers,
  SaveModelAlias,
} from "../lib/apiClient";

const emptyForm = () => ({
  id: "",
  name: "",
  supplier_id: "",
  upstream_protocol: "",
  model: "",
  enabled: true,
});

const protocolOptions = [
  { label: "Anthropic", value: "anthropic" },
  { label: "OpenAI Chat", value: "openai-chat" },
  { label: "OpenAI Responses", value: "openai-responses" },
];

const getModelName = (model) => String(model?.name || "").trim();

export const useModelAliasesStore = defineStore("modelAliases", {
  state: () => ({
    loading: false,
    saving: false,
    deleting: "",
    fetchingModels: false,
    error: "",
    items: [],
    suppliers: [],
    form: emptyForm(),
  }),
  getters: {
    enabledCount(state) {
      return state.items.filter((item) => item.enabled).length;
    },
    supplierCount(state) {
      const seen = new Set();
      state.items.forEach((item) => {
        if (item.supplier_id) {
          seen.add(item.supplier_id);
        }
      });
      return seen.size;
    },
    supplierOptions(state) {
      return state.suppliers.map((supplier) => ({
        label: `${supplier.name} (${supplier.protocol})`,
        value: supplier.id,
      }));
    },
    selectedSupplier(state) {
      return (
        state.suppliers.find(
          (supplier) => supplier.id === state.form.supplier_id,
        ) || null
      );
    },
    modelOptions(state) {
      const supplier = state.suppliers.find(
        (s) => s.id === state.form.supplier_id,
      );
      if (!supplier || !supplier.models?.length) {
        return [];
      }
      return supplier.models
        .map((model) => {
          const name = getModelName(model);
          if (!name) {
            return null;
          }
          return {
            label: name,
            value: name,
          };
        })
        .filter(Boolean);
    },
    upstreamProtocolOptions() {
      return protocolOptions;
    },
  },
  actions: {
    async load() {
      this.loading = true;
      this.error = "";
      try {
        const [items, suppliers] = await Promise.all([
          ListModelAliases(),
          ListSuppliers(),
        ]);
        this.items = items;
        this.suppliers = suppliers;
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.loading = false;
      }
    },
    select(item) {
      this.form = {
        id: item.id,
        name: item.name,
        supplier_id: item.supplier_id || "",
        upstream_protocol: item.upstream_protocol || "",
        model: item.model,
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
        this.items = await SaveModelAlias({
          id: this.form.id,
          name: String(this.form.name || "").trim(),
          supplier_id: String(this.form.supplier_id || "").trim(),
          upstream_protocol: String(this.form.upstream_protocol || "").trim(),
          model: String(this.form.model || "").trim(),
          enabled: this.form.enabled,
        });
        this.resetForm();
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
        this.items = await DeleteModelAlias(id);
        if (this.form.id === id) {
          this.resetForm();
        }
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.deleting = "";
      }
    },
    async fetchModels(providerID) {
      this.fetchingModels = true;
      try {
        const fetched = await FetchModelsFromProvider(providerID);
        if (!fetched?.length) {
          return 0;
        }
        // Merge fetched models into the supplier's local model list.
        const supplier = this.suppliers.find((s) => s.id === providerID);
        if (!supplier) {
          return 0;
        }
        const existingNames = new Set(
          (supplier.models || []).map((m) => getModelName(m)),
        );
        const newModels = fetched
          .filter((m) => !m.exists && !existingNames.has(m.name))
          .map((m) => ({
            id: m.id,
            name: m.name,
            max_tokens: m.max_tokens || 32768,
            enabled: true,
          }));
        if (newModels.length) {
          supplier.models = [...(supplier.models || []), ...newModels];
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
