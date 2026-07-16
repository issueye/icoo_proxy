import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

import {
  DeleteCatalogModel,
  ListModelCatalog,
  SaveCatalogModel,
} from "../lib/apiClient";
import { useModelCatalogStore } from "./modelCatalog";

vi.mock("../lib/apiClient", () => ({
  DeleteCatalogModel: vi.fn(),
  ListModelCatalog: vi.fn(),
  SaveCatalogModel: vi.fn(),
}));

describe("model catalog store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("derives display options and the custom model count", () => {
    const store = useModelCatalogStore();
    store.items = [
      { id: "gpt-4.1", name: "GPT-4.1", family: "OpenAI", built_in: true },
      { id: "local", name: "Local Model", family: "", built_in: false },
    ];

    expect(store.options).toEqual([
      { label: "GPT-4.1 \u00b7 OpenAI", value: "gpt-4.1" },
      { label: "Local Model", value: "local" },
    ]);
    expect(store.customCount).toBe(1);
  });

  it("normalizes missing optional fields when selecting a model", () => {
    const store = useModelCatalogStore();

    store.select({ id: "minimal", name: "Minimal" });

    expect(store.form).toEqual({
      id: "minimal",
      name: "Minimal",
      family: "",
      icon: "custom",
      max_tokens: 32768,
      description: "",
    });
  });

  it("reloads the catalog and resets the form after a successful save", async () => {
    const store = useModelCatalogStore();
    store.form = {
      id: "custom-1",
      name: "Custom One",
      family: "Local",
      icon: "custom",
      max_tokens: 8192,
      description: "test model",
    };
    const submitted = { ...store.form };
    SaveCatalogModel.mockResolvedValue({ id: "custom-1" });
    ListModelCatalog.mockResolvedValue([{ ...submitted, built_in: false }]);

    await store.save();

    expect(SaveCatalogModel).toHaveBeenCalledWith(submitted);
    expect(ListModelCatalog).toHaveBeenCalledOnce();
    expect(store.items).toEqual([{ ...submitted, built_in: false }]);
    expect(store.form).toEqual({
      id: "",
      name: "",
      family: "",
      icon: "custom",
      max_tokens: 32768,
      description: "",
    });
    expect(store.saving).toBe(false);
    expect(store.error).toBe("");
  });

  it("keeps current data and exposes API errors without leaving busy state set", async () => {
    const store = useModelCatalogStore();
    store.items = [{ id: "existing", name: "Existing" }];
    ListModelCatalog.mockRejectedValue(new Error("catalog unavailable"));

    await store.load();

    expect(store.items).toEqual([{ id: "existing", name: "Existing" }]);
    expect(store.error).toBe("catalog unavailable");
    expect(store.loading).toBe(false);
    expect(DeleteCatalogModel).not.toHaveBeenCalled();
  });
});
