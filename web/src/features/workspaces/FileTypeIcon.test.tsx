import { fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { FileTypeIcon } from "./FileTypeIcon";

afterEach(() => document.documentElement.classList.remove("dark"));
it("主题与目录展开改变图片，保留装饰语义和固定大小", async () => {
  const view = render(<FileTypeIcon path="config.toml" />);
  const image = view.container.querySelector("img")!;
  expect(image).toHaveAttribute("src", "/material-icons/toml_light.svg");
  expect(image).toHaveAttribute("alt", ""); expect(image).toHaveAttribute("aria-hidden", "true");
  document.documentElement.classList.add("dark");
  await waitFor(() => expect(image).toHaveAttribute("src", "/material-icons/toml.svg"));
  view.rerender(<FileTypeIcon path="src" kind="directory" expanded />);
  expect(image).toHaveAttribute("src", "/material-icons/folder-src-open.svg");
});
it("加载错误只回退一次，改名后可以重新解析图标", () => {
  const view = render(<FileTypeIcon path="file.rs" />); const image = view.container.querySelector("img")!;
  fireEvent.error(image); const fallback = image.getAttribute("src");
  expect(fallback).toBe("/material-icons/file.svg"); fireEvent.error(image); expect(image).toHaveAttribute("src", fallback!);
  view.rerender(<FileTypeIcon path="file.py" />); expect(image).toHaveAttribute("src", "/material-icons/python.svg");
});
