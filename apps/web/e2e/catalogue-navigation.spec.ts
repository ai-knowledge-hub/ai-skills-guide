import { expect, test } from "@playwright/test";

test("opening a result preserves pending search without cancelling detail navigation", async ({ page }) => {
  // Hold the search debounce and detail response independently so the URL
  // update can be exercised while navigation is in flight, without sleeps.
  await page.addInitScript(() => {
    const setTimeout = window.setTimeout.bind(window);
    const clearTimeout = window.clearTimeout.bind(window);
    const pending = new Map<number, () => void>();
    Object.defineProperty(window, "setTimeout", { value: (handler: TimerHandler, delay?: number, ...args: unknown[]) => {
      if (delay !== 180 || typeof handler !== "function") return setTimeout(handler, delay, ...args);
      const id = setTimeout(() => {}, 60000);
      pending.set(id, () => handler(...args));
      return id;
    } });
    Object.defineProperty(window, "clearTimeout", { value: (id?: number) => {
      if (id !== undefined) pending.delete(id);
      clearTimeout(id);
    } });
    Object.defineProperty(window, "catalogueTimers", { value: {
      count: () => pending.size,
      flush: () => {
        const callbacks = [...pending.entries()];
        pending.clear();
        for (const [id, callback] of callbacks) { clearTimeout(id); callback(); }
      },
    } });
  });
  let releaseDetail!: () => void;
  const detailGate = new Promise<void>((resolve) => { releaseDetail = resolve; });
  const path = "/plugins/marketing/performance-reporting-plugin";
  await page.route(`**${path}*`, async (route) => {
    await detailGate;
    await route.continue();
  });
  try {
    await page.goto("/plugins");
    await page.getByRole("searchbox", { name: "Search the catalogue" }).fill("Performance Reporting Plugin");
    await expect.poll(() => page.evaluate(() => (window as unknown as {
      catalogueTimers: { count: () => number };
    }).catalogueTimers.count())).toBe(1);
    await page.getByRole("link", { name: /Performance Reporting Plugin/ }).click();
    await page.evaluate(() => (window as unknown as {
      catalogueTimers: { flush: () => void };
    }).catalogueTimers.flush());
    releaseDetail();
    await expect(page).toHaveURL(path);
    await expect(page.getByRole("heading", { name: "Performance Reporting Plugin", exact: true })).toBeVisible();
    await page.goBack();
    await expect(page).toHaveURL(/\/plugins\?q=Performance\+Reporting\+Plugin$/);
    await expect(page.getByRole("searchbox")).toHaveValue("Performance Reporting Plugin");
  } finally {
    releaseDetail();
  }
});
