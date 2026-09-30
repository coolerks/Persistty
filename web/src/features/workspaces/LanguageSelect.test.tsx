import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { LanguageSelect } from "./LanguageSelect";

it("键盘选择 SQL 方言并返回自动识别", async () => {
  const user = userEvent.setup(); const onChange = vi.fn();
  const view = render(<LanguageSelect mode={undefined} detected="sql" onChange={onChange} />);
  const trigger = screen.getByRole("combobox", { name: "语言模式" });
  trigger.focus(); await user.keyboard("{Enter}");
  await user.click(await screen.findByRole("option", { name: /^PostgreSQL$/ }));
  expect(onChange).toHaveBeenLastCalledWith("pgsql");
  view.rerender(<LanguageSelect mode="pgsql" detected="sql" onChange={onChange} />);
  await user.click(trigger); await user.click(await screen.findByRole("option", { name: "自动识别" }));
  expect(onChange).toHaveBeenLastCalledWith(undefined);
});
