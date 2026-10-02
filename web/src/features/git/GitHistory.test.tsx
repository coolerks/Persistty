import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { GitHistory } from "./GitHistory";

const commit = { id: "a".repeat(40), subject: "First commit", parents: [], author: "Fixture", date: "2026-10-02T00:00:00Z" };
const history = { head: commit.id, items: [commit], next_offset: 50 };
const props = { history, detail: null, expanded: "", branch: "main", view: "list" as const, pending: false, onExpand: vi.fn(), onParent: vi.fn(), onCompare: vi.fn() };
afterEach(cleanup);

function fixture(onMore: () => Promise<boolean>) {
  const component = (pending = false, next = history) => <div className="git-section-content" data-testid="history-scroller"><GitHistory {...props} pending={pending} history={next} onMore={onMore} /></div>;
  const rendered = render(component());
  const scroller = screen.getByTestId("history-scroller");
  Object.defineProperties(scroller, { scrollHeight: { configurable: true, value: 1000 }, clientHeight: { configurable: true, value: 200 } });
  const scroll = (top: number) => { scroller.scrollTop = top; fireEvent.scroll(scroller); };
  return { ...rendered, component, scroller, scroll };
}

test("只在历史区域滚动到底时加载，同一未完成请求不重复触发", async () => {
  let finish: (value: boolean) => void = () => {};
  const load = vi.fn(() => new Promise<boolean>(resolve => { finish = resolve; }));
  const view = fixture(load);
  expect(screen.queryByRole("button", { name: "加载更多提交" })).not.toBeInTheDocument();
  expect(load).not.toHaveBeenCalled();
  view.scroll(300); fireEvent.scroll(window); expect(load).not.toHaveBeenCalled();
  view.scroll(790); view.scroll(800); expect(load).toHaveBeenCalledTimes(1);
  await act(async () => { finish(true); });
  view.rerender(view.component(false, { ...history, next_offset: 100 }));
  expect(load).toHaveBeenCalledTimes(1);
  view.scroll(800); expect(load).toHaveBeenCalledTimes(2);
  await act(async () => { finish(true); });
  view.rerender(view.component(false, { ...history, next_offset: -1 }));
  view.scroll(800); expect(load).toHaveBeenCalledTimes(2);
});

test("请求失败不在底部重复重试，离开底部再滚回可重新加载", async () => {
  const load = vi.fn().mockResolvedValueOnce(false).mockResolvedValue(true);
  const view = fixture(load);
  await act(async () => { view.scroll(800); });
  await act(async () => { view.scroll(795); view.scroll(800); });
  expect(load).toHaveBeenCalledTimes(1);
  view.scroll(300);
  await act(async () => { view.scroll(800); });
  expect(load).toHaveBeenCalledTimes(2);
});

test("前台请求或隐藏区域不加载，卸载后移除监听", async () => {
  const load = vi.fn().mockResolvedValue(true), view = fixture(load);
  view.rerender(view.component(true)); view.scroll(800); expect(load).not.toHaveBeenCalled();
  view.rerender(view.component());
  Object.defineProperty(view.scroller, "clientHeight", { configurable: true, value: 0 });
  view.scroll(1000); expect(load).not.toHaveBeenCalled();
  Object.defineProperty(view.scroller, "clientHeight", { configurable: true, value: 200 });
  await act(async () => { view.scroll(800); }); expect(load).toHaveBeenCalledTimes(1);
  view.unmount(); fireEvent.scroll(view.scroller); expect(load).toHaveBeenCalledTimes(1);
});

test("合并提交通过右键选择父提交，收起时选择默认父也能触发读取", async () => {
  const parents = ["b".repeat(40), "c".repeat(40)], merge = { ...commit, parents }, onParent = vi.fn();
  const mergedHistory = { ...history, items: [merge] }, onMore = vi.fn().mockResolvedValue(true);
  const view = render(<GitHistory {...props} history={mergedHistory} expanded={commit.id} detail={{ commit: merge, parent_id: parents[0]!, files: ["src/file.ts"] }} onParent={onParent} onMore={onMore} />);
  expect(screen.queryByRole("combobox", { name: "比较父提交" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "src/file.ts" }).closest(".git-commit-files")).not.toBeNull();
  fireEvent.contextMenu(screen.getByRole("button", { name: /^First commit/ }), { clientX: 10, clientY: 10 });
  expect(await screen.findByRole("menuitemradio", { name: "父提交 1 · bbbbbbbbbbbb" })).toHaveAttribute("aria-checked", "true");
  await userEvent.click(screen.getByRole("menuitemradio", { name: "父提交 2 · cccccccccccc" }));
  expect(onParent).toHaveBeenLastCalledWith(commit.id, parents[1]);
  expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  view.rerender(<GitHistory {...props} history={mergedHistory} onParent={onParent} onMore={onMore} />);
  fireEvent.contextMenu(screen.getByRole("button", { name: /^First commit/ }), { clientX: 10, clientY: 10 });
  await userEvent.click(await screen.findByRole("menuitemradio", { name: "父提交 1 · bbbbbbbbbbbb" }));
  expect(onParent).toHaveBeenLastCalledWith(commit.id, parents[0]); expect(onParent).toHaveBeenCalledTimes(2);
});
