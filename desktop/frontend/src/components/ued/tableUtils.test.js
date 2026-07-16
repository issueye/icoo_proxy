import { describe, expect, it } from "vitest";
import {
  buildPageItems,
  clampPage,
  formatCellValue,
  normalizeAlign,
  normalizeFixed,
  normalizePositiveInteger,
  normalizeSize,
  withStickyOffsets,
} from "./tableUtils";

describe("tableUtils", () => {
  it("normalizes sizes and aliases", () => {
    expect(normalizeSize("small")).toBe("sm");
    expect(normalizeSize("middle")).toBe("md");
    expect(normalizeSize("lg")).toBe("lg");
    expect(normalizeSize("weird")).toBe("md");
  });

  it("clamps pages", () => {
    expect(clampPage(0, 5)).toBe(1);
    expect(clampPage(9, 5)).toBe(5);
    expect(normalizePositiveInteger(-1, 10)).toBe(10);
  });

  it("formats empty cells", () => {
    expect(formatCellValue(null)).toBe("-");
    expect(formatCellValue("")).toBe("-");
    expect(formatCellValue(0)).toBe("0");
  });

  it("builds compact page lists", () => {
    const items = buildPageItems(5, 20);
    expect(items.some((i) => i.type === "ellipsis")).toBe(true);
    expect(items.find((i) => i.page === 5)).toBeTruthy();
  });

  it("computes sticky offsets", () => {
    const cols = withStickyOffsets([
      { key: "a", fixed: "left", width: "80px" },
      { key: "b", fixed: "", width: "120px" },
      { key: "c", fixed: "right", width: "100px" },
    ]);
    expect(cols[0].stickyStyle.left).toBe("0px");
    expect(cols[0].isStickyLeftLast).toBe(true);
    expect(cols[2].stickyStyle.right).toBe("0px");
    expect(cols[2].isStickyRightFirst).toBe(true);
    expect(normalizeAlign("right")).toBe("right");
    expect(normalizeFixed(true)).toBe("left");
  });
});
