import { defineStore } from "pinia";
import { DeleteCatalogModel, ListModelCatalog, SaveCatalogModel } from "../lib/apiClient";

const emptyForm = () => ({
  id: "",
  name: "",
  family: "",
  icon: "custom",
  max_tokens: 32768,
  description: "",
});

export const useModelCatalogStore = defineStore("modelCatalog", {
  state: () => ({
    items: [],
    loading: false,
    saving: false,
    deleting: "",
    error: "",
    form: emptyForm(),
  }),
  getters: {
    options: (state) => state.items.map((item) => ({
      label: item.family ? `${item.name} · ${item.family}` : item.name,
      value: item.id,
    })),
    customCount: (state) => state.items.filter((item) => !item.built_in).length,
  },
  actions: {
    async load() {
      this.loading = true;
      this.error = "";
      try {
        this.items = await ListModelCatalog();
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
        family: item.family || "",
        icon: item.icon || "custom",
        max_tokens: item.max_tokens || 32768,
        description: item.description || "",
      };
    },
    resetForm() {
      this.form = emptyForm();
    },
    async save() {
      this.saving = true;
      this.error = "";
      try {
        await SaveCatalogModel(this.form);
        await this.load();
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
        await DeleteCatalogModel(id);
        await this.load();
      } catch (error) {
        this.error = error?.message || String(error);
      } finally {
        this.deleting = "";
      }
    },
  },
});
