import { expect, it } from "vitest";
import { isVerticalWheel, wheelPixels } from "./terminal-scrolling";

it("像素/行/页滚轮保持原始小数位移，不把每个微小事件放大到整行", () => {
  expect(wheelPixels(-64, 0, 16, 384)).toBe(-64);
  expect(wheelPixels(0.5, 0, 16, 384)).toBe(0.5);
  expect(wheelPixels(-3, 1, 16, 384)).toBe(-48);
  expect(wheelPixels(1, 2, 16, 384)).toBe(384);
  expect(wheelPixels(0, 0, 16, 384)).toBe(0);
  expect(wheelPixels(NaN, 0, 16, 384)).toBe(0);
});

it("横向主轴及纵向噪声不切换历史，垂直和等幅斜向可滚动", () => {
  expect(isVerticalWheel(-180, -0.25)).toBe(false);
  expect(isVerticalWheel(120, 0)).toBe(false);
  expect(isVerticalWheel(0, 0)).toBe(false);
  expect(isVerticalWheel(0.25, -20)).toBe(true);
  expect(isVerticalWheel(-20, 20)).toBe(true);
  expect(isVerticalWheel(0, NaN)).toBe(false);
});
