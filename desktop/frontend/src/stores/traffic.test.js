import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

import { GetTrafficPage } from "../lib/apiClient";
import { useTrafficStore } from "./traffic";

vi.mock("../lib/apiClient", () => ({
  ClearTrafficRequests: vi.fn(),
  GetTrafficPage: vi.fn(),
}));

describe("traffic store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("loads cancellation metrics and applies protocol filters from page one", async () => {
    GetTrafficPage.mockResolvedValueOnce({
      items: [{ id: 7, status_code: 499 }],
      total: 1,
      page: 1,
      page_size: 8,
      total_requests: 12,
      success_count: 8,
      error_count: 2,
      canceled_count: 2,
      average_latency: 125,
      protocol_options: ["all", "openai"],
    }).mockResolvedValueOnce({
      items: [],
      total: 0,
      page: 1,
      page_size: 8,
      protocol_options: ["all", "openai"],
    });
    const store = useTrafficStore();

    await store.load();

    expect(store.canceledCount).toBe(2);
    expect(store.successCount).toBe(8);
    expect(store.errorCount).toBe(2);
    expect(store.requests).toEqual([{ id: 7, status_code: 499 }]);

    await store.setFilter("openai");

    expect(GetTrafficPage).toHaveBeenLastCalledWith(1, 8, "openai");
    expect(store.filter).toBe("openai");
    expect(store.page).toBe(1);
    expect(store.loading).toBe(false);
  });

  it("refetches the last valid page when the current page becomes empty", async () => {
    GetTrafficPage.mockResolvedValueOnce({
      items: [],
      total: 17,
      page: 4,
      page_size: 8,
    }).mockResolvedValueOnce({
      items: [{ id: 17 }],
      total: 17,
      page: 3,
      page_size: 8,
    });
    const store = useTrafficStore();

    await store.fetchPage({ page: 4, pageSize: 8 });

    expect(GetTrafficPage).toHaveBeenNthCalledWith(1, 4, 8, "all");
    expect(GetTrafficPage).toHaveBeenNthCalledWith(2, 3, 8, "all");
    expect(store.page).toBe(3);
    expect(store.requests).toEqual([{ id: 17 }]);
  });
});
