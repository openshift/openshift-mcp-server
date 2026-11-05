import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { inflateSync } from "node:zlib";

// The shared runner preserves extensions/openshift under its Playwright directory.
const assets = new URL("../../../../../pkg/mcpapps/openshift/", import.meta.url);
const template = readFileSync(new URL("app.html", assets), "utf8");
const runtime = readFileSync(new URL("runtime.js", assets), "utf8");
const presentation = readFileSync(new URL("presentation.js", assets), "utf8");
const styles = readFileSync(new URL("presentation.css", assets), "utf8");
const origin = "https://mcp-apps.test";

async function openApp(page, kind = "table", name = "Namespaces list") {
  await page.route(origin + "/", route => route.fulfill({ contentType: "text/html", body: "<!doctype html>" }));
  await page.goto(origin + "/");
  await page.setContent('<iframe style="width:800px;height:600px;border:0"></iframe>');
  await page.evaluate(origin => {
    const source = document.querySelector("iframe").contentWindow;
    window.messages = [];
    window.addEventListener("message", event => {
      if (event.source !== source || event.origin !== origin) return;
      window.messages.push(event.data);
    });
  }, origin);
  const app = page.frames()[1];
  // Use callbacks so $ sequences in the generated bundle remain literal.
  let html = template;
  for (const [key, value] of Object.entries({ KIND: kind, NAME: name, SCRIPT: runtime,
    PRESENTATION_SCRIPT: presentation, PRESENTATION_STYLES: styles })) html = html.replace("{{" + key + "}}", () => value);
  await app.setContent(html);
  const card = app.getByRole("article");
  await expect(card).toBeVisible();
  await expect(card).toHaveAttribute("aria-busy", "true");
  await expect(card.getByRole("img", { name: "Red Hat", exact: true })).toBeVisible();
  await expect(card.locator(".title-skeleton")).toBeVisible();
  await expect(card.locator(".description-skeleton")).toBeVisible();
  await expect(app.getByRole("status")).toHaveText("Loading data…");
  expect(await app.getByRole("status").evaluate(element => element.closest('[aria-busy="true"]') === null)).toBe(true);
  return app;
}

async function send(page, message) {
  await page.evaluate(({ origin, message }) => {
    document.querySelector("iframe").contentWindow.postMessage({ jsonrpc: "2.0", ...message }, origin);
  }, { origin, message });
}
const result = (page, structuredContent) => send(page, { method: "ui/notifications/tool-result", params: { structuredContent } });

// Decode just the first pixel of a 1px screenshot, without an image dependency.
function screenshotRgb(png) {
  expect(png.readUInt32BE(16)).toBe(1);
  expect(png.readUInt32BE(20)).toBe(1);
  expect(png[24]).toBe(8);
  expect([2, 6]).toContain(png[25]);
  const chunks = [];
  for (let offset = 8; offset < png.length;) {
    const size = png.readUInt32BE(offset);
    if (png.toString("ascii", offset + 4, offset + 8) === "IDAT") chunks.push(png.subarray(offset + 8, offset + 8 + size));
    offset += size + 12;
  }
  return [...inflateSync(Buffer.concat(chunks)).subarray(1, 4)];
}

test("PatternFly renders in-card branding, real rows, namespace badges, and both sorting directions", async ({ page }, testInfo) => {
  const errors = [];
  page.on("pageerror", error => errors.push(error.message));
  const app = await openApp(page);
  await result(page, { columns: ["Name", "Namespace", "Status", "Age"], items: [
    { Name: "z", Namespace: "team-a", Status: "Running", Age: "1h" },
    { Name: "a", Namespace: "team-b", Status: "Pending", Age: "10m" },
  ] });
  await expect(app.getByRole("article")).toHaveClass(/pf-v6-c-card/);
  await expect(app.locator("table")).toHaveClass(/pf-v6-c-table/);
  await expect(app.getByRole("heading", { level: 1 })).toHaveText("Namespaces list");
  await expect(app.getByRole("article")).not.toHaveAttribute("aria-busy", "true");
  await expect(app.locator("main > .identity")).toHaveCount(0);
  await expect(app.locator(".eyebrow, .product")).toHaveCount(0);
  await expect(app.locator(".result-description")).toHaveText("2 rows returned · Kubernetes resource inventory");
  await expect(app.locator("tbody tr:last-child")).toHaveCSS("border-bottom-style", "none");
  await expect(app.locator("tbody tr td:first-child")).toHaveText(["a", "z"]);
  await expect(app.locator(".namespace .pf-v6-c-label")).toHaveText(["NS", "NS"]);
  await app.getByRole("button", { name: "Name", exact: true }).click();
  await expect(app.locator("tbody tr td:first-child")).toHaveText(["z", "a"]);
  await app.getByRole("button", { name: "Age", exact: true }).click();
  await expect(app.locator("tbody tr td:last-child")).toHaveText(["1h", "10m"]);
  await page.screenshot({ path: testInfo.outputPath("table.png") });
  expect(errors).toEqual([]);
});

test("PatternFly uses host themes and transparent surroundings, stays offline, and adapts within narrow cards", async ({ page }) => {
  const app = await openApp(page);
  const requests = [];
  await page.route("**/*", route => { requests.push(route.request().url()); return route.abort(); });
  await page.locator("iframe").evaluate(frame => { frame.style.width = "320px"; });
  await send(page, { id: 1, result: { hostContext: { theme: "dark" } } });
  await expect(app.locator("html")).toHaveClass(/pf-v6-theme-dark/);
  await expect(app.locator(".logo-dark")).toBeVisible();
  await result(page, { items: [{ Name: "default", Status: "Active", Age: "2h", Labels: "test=value".repeat(30), apiVersion: "v1", kind: "Namespace" }] });
  await expect(app.locator("tbody tr")).toHaveCount(1);
  expect(await app.locator(".table-scroll").evaluate(element => element.scrollWidth > element.clientWidth)).toBe(true);
  expect(await app.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBe(0);
  for (const width of [320, 480, 600, 800]) {
    await page.locator("iframe").evaluate((frame, width) => { frame.style.width = width + "px"; }, width);
    expect(await app.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    const aligned = await app.getByRole("article").evaluate(card => {
      const identity = card.querySelector(".identity").getBoundingClientRect();
      const content = card.querySelector(".result-content").getBoundingClientRect();
      return Math.abs(identity.left - content.left) < 1 && Math.abs(identity.right - content.right) < 1;
    });
    expect(aligned).toBe(true);
  }
  // All views use this shared shell. Check its surface as part of host theming.
  await expect(app.locator("html")).toHaveClass(/pf-mcp-app-document/);
  await app.locator("html").evaluate(element => {
    element.style.setProperty("--pf-t--global--background--image--default", "linear-gradient(red, blue)");
  });
  let previousCardColor;
  for (const theme of ["dark", "light"]) {
    await page.evaluate(theme => {
      document.documentElement.style.colorScheme = theme === "light" ? "dark" : "light";
      document.body.style.background = "rgb(18, 52, 86)";
      // The embedding element is host-owned and must agree with the app.
      document.querySelector("iframe").style.colorScheme = theme;
    }, theme);
    await send(page, { method: "ui/notifications/host-context-changed", params: { theme } });
    await expect(app.locator("html")).toHaveCSS("color-scheme", theme);
    const surfaces = await app.evaluate(() => [document.documentElement, document.body,
      document.getElementById("app"), document.querySelector("main")].map(element => ({
      color: getComputedStyle(element).backgroundColor, image: getComputedStyle(element).backgroundImage,
    })));
    expect(surfaces).toEqual(Array(4).fill({ color: "rgba(0, 0, 0, 0)", image: "none" }));
    const card = app.getByRole("article");
    await expect(card).not.toHaveCSS("background-color", "rgba(0, 0, 0, 0)");
    if (previousCardColor) await expect(card).not.toHaveCSS("background-color", previousCardColor);
    previousCardColor = await card.evaluate(element => getComputedStyle(element).backgroundColor);
    const frame = await page.locator("iframe").boundingBox();
    const pixel = await page.screenshot({ clip: { x: Math.ceil(frame.x) + 4, y: Math.ceil(frame.y) + 4, width: 1, height: 1 }, scale: "css" });
    expect(screenshotRgb(pixel)).toEqual([18, 52, 86]);
  }
  await expect(app.locator("html")).not.toHaveClass(/pf-v6-theme-dark/);
  await expect(app.locator(".logo-light")).toBeVisible();
  expect(requests).toEqual([]);
});

test("PatternFly metrics retain numeric quantity sorting and recover after errors", async ({ page }) => {
  const app = await openApp(page, "metrics", "Pod metrics");
  await send(page, { id: "app-ping", method: "ping" });
  await expect.poll(() => page.evaluate(() => window.messages.find(message => message.id === "app-ping")?.result)).toEqual({});
  await send(page, { method: "ui/notifications/tool-cancelled" });
  await expect(app.getByRole("status")).toHaveText("Tool execution cancelled.");
  await result(page, { items: [] });
  await expect(app.getByRole("status")).toHaveText("No metrics found.");
  await send(page, { method: "ui/notifications/tool-result", params: { isError: true, content: [{ type: "text", text: "Forbidden" }] } });
  await expect(app.getByRole("status")).toHaveText("Unable to load data: Forbidden");
  await result(page, { columns: ["Name", "CPU", "Memory"], items: [{ Name: "a", CPU: "1", Memory: "1Gi" }, { Name: "b", CPU: "20m", Memory: "20Mi" }] });
  await app.getByRole("button", { name: "CPU", exact: true }).click();
  await expect(app.locator("tbody td:nth-child(2)")).toHaveText(["20m", "1"]);
});

test("PatternFly resource details expand source data and report content-size changes", async ({ page }) => {
  const app = await openApp(page, "resource", "Pod details");
  await send(page, { id: 1, result: { hostContext: { theme: "light" } } });
  const resource = { apiVersion: "v1", kind: "Pod", metadata: { name: "web", namespace: "team-a" },
    spec: { containers: [{ name: "app", image: "nginx" }] },
    status: { phase: "Running", conditions: [{ type: "Ready", status: "False", reason: "ContainersNotReady" }] } };
  await result(page, resource);
  await expect(app.getByRole("heading", { level: 1 })).toHaveText("Pod web");
  await expect(app.getByRole("region", { name: "Overview", exact: true })).toContainText("team-a");
  await expect(app.getByRole("region", { name: "Containers", exact: true })).toContainText("nginx");
  await expect.poll(() => page.evaluate(() => window.messages.filter(m => m.method === "ui/notifications/size-changed").length)).toBeGreaterThan(0);
  const before = await app.evaluate(() => document.body.getBoundingClientRect().height);
  await app.getByRole("button", { name: "Raw resource", exact: true }).click();
  await expect(app.locator("pre")).toHaveText(JSON.stringify(resource, null, 2));
  await expect.poll(() => page.evaluate(() => window.messages.filter(m => m.method === "ui/notifications/size-changed").at(-1)?.params.height || 0)).toBeGreaterThan(before);
});
