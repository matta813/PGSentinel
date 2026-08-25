import { chromium } from "@playwright/test";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

const frontendDir = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const output = path.resolve(
  frontendDir,
  "..",
  "docs",
  "assets",
  "pgsentinel-operator-controls.png",
);
const baseURL = "http://127.0.0.1:4173";
const vite = spawn(
  "npm",
  ["run", "dev", "--", "--host", "127.0.0.1", "--port", "4173"],
  { cwd: frontendDir, stdio: "ignore" },
);
async function ready() {
  for (let i = 0; i < 60; i++) {
    try {
      if ((await fetch(baseURL)).ok) return;
    } catch {
      /* starting */
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error("Vite did not start");
}
try {
  await ready();
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({
    viewport: { width: 1440, height: 1000 },
  });
  await page.addInitScript(() => localStorage.setItem("theme", "dark"));
  await page.route("**/api/v1/**", async (route) => {
    const pathname = new URL(route.request().url()).pathname;
    let body;
    if (pathname === "/api/v1/auth/session")
      body = {
        authenticated: true,
        username: "admin",
        mustChangePassword: false,
      };
    else if (pathname === "/api/v1/version")
      body = { version: "0.6.0", commit: "demo" };
    else if (pathname === "/api/v1/servers")
      body = [
        {
          id: "primary",
          name: "Production primary",
          status: "healthy",
          tags: ["production"],
        },
      ];
    else if (
      pathname === "/api/v1/notifications" ||
      pathname === "/api/v1/notification-routes" ||
      pathname === "/api/v1/notification-deliveries"
    )
      body = [];
    else if (pathname === "/api/v1/maintenance-windows")
      body = [
        {
          id: "window",
          description: "Planned primary failover",
          serverId: "primary",
          category: "Replication",
          startsAt: "2026-08-25T08:00:00Z",
          endsAt: "2026-08-25T10:00:00Z",
          state: "active",
        },
      ];
    else if (pathname === "/api/v1/suppressions")
      body = [
        {
          id: "suppression",
          ruleId: "blocking-queries",
          serverId: "primary",
          reason: "Expected lock test",
          expiresAt: "2026-08-25T09:30:00Z",
          state: "active",
        },
      ];
    else if (pathname === "/api/v1/threshold-overrides")
      body = {
        items: [
          {
            id: "threshold",
            ruleId: "standby-replay-lag",
            scopeType: "server",
            scopeValue: "primary",
            value: 120,
            reason: "Intentionally delayed reporting replica",
          },
        ],
        specs: {
          "standby-replay-lag": {
            label: "Replica replay lag",
            min: 10,
            max: 86400,
            default: 60,
            unit: "seconds",
          },
          "blocking-queries": {
            label: "Blocking duration",
            min: 5,
            max: 3600,
            default: 60,
            unit: "seconds",
          },
        },
      };
    await route.fulfill({
      status: body ? 200 : 404,
      contentType: "application/json",
      body: JSON.stringify(body ?? { error: "Not found" }),
    });
  });
  await page.goto(`${baseURL}/settings#maintenance`, {
    waitUntil: "networkidle",
  });
  await page.addStyleTag({
    content: ".sidebar,.topbar,.page-header,.settings-nav,#destinations,#routing,#delivery-history{display:none!important}.app-shell,.settings-layout{display:block}.page{padding:0;max-width:760px}.operator-controls{margin:0}",
  });
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: output, fullPage: true });
  await browser.close();
} finally {
  vite.kill("SIGTERM");
}
