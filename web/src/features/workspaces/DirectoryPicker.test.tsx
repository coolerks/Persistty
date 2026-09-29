import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import { api } from "@/lib/api/client";
import { DirectoryPicker } from "./DirectoryPicker";

afterEach(() => vi.restoreAllMocks());

it("关闭目录选择器会取消当前逐层浏览请求", async () => {
  let active: AbortSignal | undefined;
  vi.spyOn(api, "directory")
    .mockResolvedValueOnce({ path: "/home", parent: "/", items: ["project"] })
    .mockImplementationOnce((_path, signal) => { active = signal; return new Promise(() => undefined); });
  const props = { onOpenChange: vi.fn(), onSelect: vi.fn() };
  const { rerender } = render(<DirectoryPicker open {...props} />);
  await userEvent.setup().click(await screen.findByRole("button", { name: "project" }));
  await waitFor(() => expect(active).toBeDefined());
  rerender(<DirectoryPicker open={false} {...props} />);
  expect(active?.aborted).toBe(true);
});
