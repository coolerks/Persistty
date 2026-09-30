import { expect, type Locator, type Page } from "@playwright/test";

export async function menuAction(page: Page, runtime: Locator, name: string) {
  const id = await runtime.getAttribute("data-terminal-id");
  const tab = page.locator(`.terminal-tab[data-terminal-id="${id}"]`);
  const more = tab.getByRole("button", { name: /更多$/ });
  if (await more.isVisible()) await more.click();
  else await tab.click({ button: "right" });
  await page.getByRole("menuitem", { name, exact: true }).click();
}

export async function controlled(runtime: Locator, timeout = 10000) {
  await expect(runtime).toHaveAttribute("data-connection", "connected", { timeout });
  await expect(runtime).toHaveAttribute("data-role", "controller", { timeout });
}
