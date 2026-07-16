// @vitest-environment jsdom

import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import UButton from "./UButton.vue";

describe("UButton", () => {
  it("emits clicks while enabled", async () => {
    const wrapper = mount(UButton, {
      slots: { default: "Save settings" },
    });

    await wrapper.get("button").trigger("click");

    expect(wrapper.text()).toContain("Save settings");
    expect(wrapper.emitted("click")).toHaveLength(1);
  });

  it("disables interaction and shows a spinner while loading", async () => {
    const wrapper = mount(UButton, {
      props: { loading: true },
      slots: { default: "Save settings" },
    });
    const button = wrapper.get("button");

    expect(button.attributes("disabled")).toBeDefined();
    expect(wrapper.find(".ued-button__spinner").exists()).toBe(true);

    await button.trigger("click");
    expect(wrapper.emitted("click")).toBeUndefined();
  });
});
