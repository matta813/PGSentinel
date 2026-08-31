import { chromium } from "@playwright/test";
import { spawn } from "node:child_process";
import { mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";

const frontendDir = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const outputDir = path.resolve(frontendDir, "..", "docs", "assets");
const baseURL = "http://127.0.0.1:4173";
const vite = spawn(
  "npm",
  ["run", "dev", "--", "--host", "127.0.0.1", "--port", "4173"],
  { cwd: frontendDir, stdio: "ignore" },
);

const now = "2026-08-31T08:30:00.000Z";
const server = {
  id: "prod-primary",
  name: "Production primary · checkout EU west",
  host: "postgres-primary.internal",
  port: 5432,
  user: "pgsentinel",
  sslMode: "verify-full",
  version: "16.4",
  status: "healthy",
  lastConnectedAt: now,
  tags: ["production", "primary"],
};
const degradedServer = {
  id: "analytics-replica",
  name: "Analytics replica",
  host: "postgres-analytics.internal",
  port: 5432,
  user: "pgsentinel",
  sslMode: "verify-full",
  version: "16.4",
  status: "degraded",
  lastConnectedAt: now,
  lastError:
    "WAL statistics collection failed; cached evidence is being preserved.",
  tags: ["production", "replica"],
};
const findings = [
  {
    id: "long-transaction",
    serverId: server.id,
    database: "payments",
    resource: "PID 18422",
    severity: "CRITICAL",
    category: "transactions",
    title: "Long-running transaction is holding back cleanup",
    summary:
      "A transaction has remained open for 47 minutes and may prevent vacuum from reclaiming dead tuples.",
    cause:
      "The session is idle in transaction after a reporting query completed.",
    impact:
      "Old row versions remain visible, increasing table bloat and transaction ID pressure while the transaction stays open.",
    evidence: [
      { label: "Transaction age", value: "47", unit: "minutes" },
      { label: "Session state", value: "idle in transaction" },
      { label: "Database", value: "payments" },
    ],
    suggestions: [
      {
        title: "Identify the owning workload",
        detail:
          "Inspect PID 18422 and confirm whether the client still needs the transaction.",
      },
      {
        title: "Check cleanup pressure",
        detail:
          "Review dead tuples and autovacuum progress on affected tables before intervening.",
      },
    ],
    confidence: "HIGH",
    status: "active",
    startedAt: now,
    updatedAt: now,
  },
  {
    id: "connection-pressure",
    serverId: server.id,
    database: "payments",
    severity: "HIGH",
    category: "connections",
    title: "Connection capacity is close to exhaustion",
    summary:
      "Active and idle sessions are using 91% of available PostgreSQL connections.",
    cause: "Connection usage has crossed the high-pressure threshold.",
    impact:
      "New clients may be rejected if demand increases or reserved capacity is consumed.",
    evidence: [
      { label: "Connections used", value: "182 / 200" },
      { label: "Utilization", value: "91", unit: "%" },
    ],
    suggestions: [
      {
        title: "Break down sessions by application",
        detail:
          "Find unexpected pools or clients retaining more connections than intended.",
      },
    ],
    confidence: "HIGH",
    status: "active",
    startedAt: now,
    updatedAt: now,
  },
  {
    id: "vacuum-pressure",
    serverId: server.id,
    database: "orders",
    resource: "public.order_events",
    severity: "MEDIUM",
    category: "vacuum",
    title: "Table is accumulating dead tuples",
    summary:
      "Dead tuples are growing faster than the most recent autovacuum cycle is clearing them.",
    cause:
      "A write-heavy table is approaching its estimated autovacuum trigger.",
    impact: "Continued growth can increase storage use and slow scans.",
    evidence: [
      { label: "Dead tuples", value: "1.8M" },
      { label: "Vacuum progress", value: "86", unit: "%" },
    ],
    suggestions: [
      {
        title: "Review autovacuum history",
        detail:
          "Confirm workers are reaching the table and completing without cancellation.",
      },
    ],
    confidence: "MEDIUM",
    status: "active",
    startedAt: now,
    updatedAt: now,
  },
];

async function waitForVite() {
  for (let attempt = 0; attempt < 60; attempt += 1) {
    try {
      if ((await fetch(baseURL)).ok) return;
    } catch {
      /* Vite is still starting. */
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error("Vite did not start within 15 seconds");
}

try {
  await waitForVite();
  await mkdir(outputDir, { recursive: true });
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({
    viewport: { width: 1440, height: 960 },
    deviceScaleFactor: 1,
  });
  await page.clock.install({ time: new Date(now) });
  const browserErrors = [];
  let serverListUnavailable = false;
  page.on("pageerror", (error) => browserErrors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error" && !serverListUnavailable)
      browserErrors.push(message.text());
  });
  await page.addInitScript(() => localStorage.setItem("theme", "light"));
  await page.route("**/api/v1/**", async (route) => {
    const pathname = new URL(route.request().url()).pathname;
    if (serverListUnavailable && pathname === "/api/v1/servers") {
      await route.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({ error: "Server list temporarily unavailable" }),
      });
      return;
    }
    const resource = pathname.split("/").at(-1);
    const resourceBodies = {
      databases: {
        databases: [
          { Name: "payments", SizeBytes: 12884901888 },
          { Name: "customer_reporting_archive_2026", SizeBytes: 7516192768 },
        ],
      },
      queries: [
        {
          QueryID: "q1",
          Database: "payments",
          Query:
            "SELECT id, status, total_amount FROM orders WHERE customer_id = $1 ORDER BY created_at DESC LIMIT $2",
          Calls: 184220,
          MeanExecMS: 8.4,
          TotalExecMS: 1547464,
          ImpactScore: 92.4,
        },
        {
          QueryID: "q2",
          Database: "payments",
          Query:
            "UPDATE payment_attempts SET status = $1, updated_at = now() WHERE id = $2",
          Calls: 74381,
          MeanExecMS: 14.7,
          TotalExecMS: 1093399,
          ImpactScore: 76.1,
        },
        {
          QueryID: "q3",
          Database: "orders",
          Query:
            "SELECT date_trunc($1, created_at), count(*) FROM order_events WHERE created_at >= $2 GROUP BY 1",
          Calls: 921,
          MeanExecMS: 486.2,
          TotalExecMS: 447790,
          ImpactScore: 64.8,
        },
        {
          QueryID: "q4",
          Database: "payments",
          Query:
            "INSERT INTO ledger_entries (account_id, amount, currency, reference) VALUES ($1, $2, $3, $4)",
          Calls: 52194,
          MeanExecMS: 5.1,
          TotalExecMS: 266189,
          ImpactScore: 43.2,
        },
      ],
      indexes: [
        {
          Database: "payments",
          Schema: "public",
          Table: "orders",
          Index: "orders_customer_created_idx",
          SizeBytes: 1258291200,
          Scans: 982201,
          Unique: false,
          Primary: false,
        },
        {
          Database: "payments",
          Schema: "public",
          Table: "payment_attempts",
          Index: "payment_attempts_reference_idx",
          SizeBytes: 503316480,
          Scans: 0,
          Unique: false,
          Primary: false,
        },
        {
          Database: "orders",
          Schema: "public",
          Table: "order_events",
          Index: "order_events_pkey",
          SizeBytes: 2147483648,
          Scans: 4201831,
          Unique: true,
          Primary: true,
        },
      ],
      tables: [
        {
          Database: "payments",
          Schema: "public",
          Table: "orders",
          EstimatedRows: 8421900,
          TotalSize: 12884901888,
          IndexSize: 3758096384,
          LiveTuples: 8399000,
          DeadTuples: 22900,
          SeqScans: 1294,
          IndexScans: 9842201,
          LastAutovacuum: now,
          VacuumThreshold: 1684800,
          VacuumProgress: 1.4,
        },
        {
          Database: "orders",
          Schema: "public",
          Table: "order_events",
          EstimatedRows: 28410000,
          TotalSize: 33285996544,
          IndexSize: 9663676416,
          LiveTuples: 26610000,
          DeadTuples: 1800000,
          SeqScans: 1834,
          IndexScans: 12401772,
          LastAutovacuum: now,
          VacuumThreshold: 5323000,
          VacuumProgress: 33.8,
        },
        {
          Database: "payments",
          Schema: "audit",
          Table: "payment_transitions",
          EstimatedRows: 4230000,
          TotalSize: 6979321856,
          IndexSize: 2147483648,
          LiveTuples: 3850000,
          DeadTuples: 380000,
          SeqScans: 96,
          IndexScans: 645022,
          VacuumThreshold: 770050,
          VacuumProgress: 49.3,
        },
      ],
      vacuum: [
        {
          Database: "orders",
          Schema: "public",
          Table: "order_events",
          EstimatedRows: 28410000,
          TotalSize: 33285996544,
          IndexSize: 9663676416,
          LiveTuples: 26610000,
          DeadTuples: 1800000,
          SeqScans: 1834,
          IndexScans: 12401772,
          LastAutovacuum: now,
          VacuumThreshold: 2050000,
          VacuumProgress: 87.8,
        },
        {
          Database: "payments",
          Schema: "audit",
          Table: "payment_transitions",
          EstimatedRows: 4230000,
          TotalSize: 6979321856,
          IndexSize: 2147483648,
          LiveTuples: 3850000,
          DeadTuples: 380000,
          SeqScans: 96,
          IndexScans: 645022,
          VacuumThreshold: 770050,
          VacuumProgress: 49.3,
        },
      ],
      locks: [
        {
          BlockedPID: 18422,
          BlockingPID: 17201,
          DurationSeconds: 286,
          Database: "payments",
          User: "checkout",
          Application: "checkout-api",
          Query: "UPDATE orders SET status = $1 WHERE id = $2",
          BlockingQuery: "SELECT * FROM orders WHERE id = $1 FOR UPDATE",
        },
      ],
      replication: {
        InRecovery: false,
        TimelineID: 17,
        RecoveryPaused: false,
        ReplayDelaySeconds: 0,
        Standbys: [
          {
            Application: "analytics-replica",
            ClientAddress: "10.21.4.18",
            State: "streaming",
            SyncState: "async",
            ReplayLagSeconds: 1.8,
            PendingReplayBytes: 4194304,
            PendingReplayGrowthBytesPerSecond: -1024,
            ReplyAgeSeconds: 1,
          },
        ],
        Slots: [
          {
            Name: "analytics_slot",
            Type: "physical",
            Database: "",
            WALStatus: "reserved",
            Active: true,
            RetainedBytes: 16777216,
            RetainedGrowthBytesPerSecond: 0,
            InactiveSeconds: 0,
          },
        ],
      },
      wal: {
        ArchiveMode: "on",
        ArchiveConfigured: true,
        ArchivedCount: 184201,
        FailedArchiveCount: 2,
        LastArchivedWAL: "00000011000002A10000008F",
        LastFailedWAL: "000000110000029F000000E1",
        LastArchivedAt: now,
        LastFailedAt: now,
        WALBytes: 48318382080,
        GenerationBytesPerSecond: 1843200,
        WALBuffersFull: 14,
        BufferFullEventsPerSecond: 0,
        TimedCheckpoints: 1284,
        RequestedCheckpoints: 42,
        RestartpointsTimed: 0,
        RestartpointsRequested: 0,
        RestartpointsDone: 0,
        CurrentLSN: "2A1/8F04D220",
      },
    };
    const bodies = {
      "/api/v1/auth/session": {
        authenticated: true,
        username: "admin",
        role: "administrator",
        mustChangePassword: false,
      },
      "/api/v1/version": { version: "0.7.0", commit: "demo" },
      "/api/v1/overview": {
        servers: [server, degradedServer],
        problems: findings,
        counts: { CRITICAL: 1, HIGH: 1, MEDIUM: 1 },
        score: {
          overall: 68,
          categories: {
            connections: 61,
            transactions: 44,
            queries: 87,
            vacuum: 72,
            indexes: 93,
          },
        },
        freshness: {
          [server.id]: [{ state: "fresh", lastSuccessfulCollection: now }],
          [degradedServer.id]: [
            { state: "stale", lastSuccessfulCollection: now },
          ],
        },
      },
      "/api/v1/problems": findings,
      "/api/v1/incidents": [
        {
          id: "incident-1",
          serverId: server.id,
          severity: "CRITICAL",
          status: "active",
          title: "Transaction pressure affecting checkout writes",
          summary:
            "Overlapping transaction and connection evidence requires investigation.",
          startedAt: now,
          updatedAt: now,
          rationale: [
            "The findings affect the same server and database.",
            "Their observed windows overlap.",
          ],
          timeline: [
            {
              type: "finding_started",
              at: now,
              findingId: "long-transaction",
              severity: "CRITICAL",
              title: "Long-running transaction observed",
              detail: "PID 18422 remained idle in transaction.",
            },
            {
              type: "finding_started",
              at: now,
              findingId: "connection-pressure",
              severity: "HIGH",
              title: "Connection pressure observed",
              detail:
                "Connection utilization crossed the configured threshold.",
            },
          ],
          findings,
        },
      ],
      "/api/v1/servers": [server, degradedServer],
      "/api/v1/notifications": [],
      "/api/v1/notification-routes": [],
      "/api/v1/notification-deliveries": [],
      "/api/v1/maintenance-windows": [],
      "/api/v1/suppressions": [],
      "/api/v1/threshold-overrides": {
        items: [],
        specs: {
          "standby-replay-lag": {
            label: "Standby replay lag",
            min: 1,
            max: 86400,
            default: 60,
            unit: "seconds",
          },
        },
      },
      "/api/v1/rule-profiles": { items: [] },
      "/api/v1/change-events": [],
    };
    const freshness = Object.keys(resourceBodies).map((name) => ({
      serverId: server.id,
      resource: name,
      state: "fresh",
      collectedAt: now,
      lastSuccessfulCollection: now,
      ageSeconds: 18,
      expectedIntervalSeconds: 60,
      consecutiveFailures: 0,
    }));
    const body = pathname.endsWith("/freshness")
      ? freshness
      : pathname.includes(`/servers/${server.id}/`)
        ? resourceBodies[resource]
        : pathname === "/api/v1/incidents/incident-1"
          ? bodies["/api/v1/incidents"][0]
          : bodies[pathname];
    await route.fulfill({
      status: body ? 200 : 404,
      contentType: "application/json",
      body: JSON.stringify(body ?? { error: "Not found" }),
    });
  });

  await page.goto(baseURL, { waitUntil: "networkidle" });
  await page.screenshot({
    path: path.join(outputDir, "pgsentinel-overview.png"),
    fullPage: true,
  });
  await page.goto(`${baseURL}/problems?id=long-transaction`, {
    waitUntil: "networkidle",
  });
  await page.locator("#finding-long-transaction").waitFor();
  await page.screenshot({
    path: path.join(outputDir, "pgsentinel-problem-detail.png"),
    fullPage: true,
  });
  for (const [route, filename] of [
    ["queries", "pgsentinel-query-performance.png"],
    ["tables", "pgsentinel-tables.png"],
    ["indexes", "pgsentinel-index-analysis.png"],
    ["vacuum", "pgsentinel-vacuum.png"],
    ["locks", "pgsentinel-locks.png"],
    ["replication", "pgsentinel-replication.png"],
    ["wal", "pgsentinel-wal-archive.png"],
    ["incidents?id=incident-1", "pgsentinel-incidents.png"],
    ["servers", "pgsentinel-servers.png"],
    ["settings", "pgsentinel-settings.png"],
  ]) {
    await page.goto(`${baseURL}/${route}`, { waitUntil: "networkidle" });
    await page.screenshot({
      path: path.join(outputDir, filename),
      fullPage: true,
    });
  }
  serverListUnavailable = true;
  await page.goto(`${baseURL}/queries`, { waitUntil: "networkidle" });
  await page.screenshot({
    path: path.join(outputDir, "pgsentinel-server-list-unavailable.png"),
    fullPage: true,
  });
  serverListUnavailable = false;
  await page.goto(`${baseURL}/settings`, { waitUntil: "networkidle" });
  await page.getByRole("button", { name: "Switch to dark theme" }).click();
  await page.screenshot({
    path: path.join(outputDir, "pgsentinel-settings-dark.png"),
    fullPage: true,
  });
  await page.getByRole("button", { name: "Switch to light theme" }).click();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(`${baseURL}/queries`, { waitUntil: "networkidle" });
  await page.screenshot({
    path: path.join(outputDir, "pgsentinel-mobile-context.png"),
    fullPage: true,
  });
  await page.getByRole("button", { name: "Open navigation" }).click();
  await page.waitForTimeout(250);
  await page.screenshot({
    path: path.join(outputDir, "pgsentinel-mobile-navigation.png"),
    fullPage: true,
  });
  await page.setViewportSize({ width: 1280, height: 640 });
  await page.setContent(`<!doctype html><html><head><style>
    * { box-sizing: border-box } body { margin: 0; width: 1280px; height: 640px; overflow: hidden; background: #070c12; color: #f5f7fa; font-family: Inter, ui-sans-serif, system-ui, sans-serif }
    main { position: relative; width: 100%; height: 100%; padding: 92px 100px; background: radial-gradient(circle at 88% 12%, rgba(34,211,173,.16), transparent 31%), linear-gradient(135deg, #070c12 0%, #0c141e 100%) }
    main::after { content: ''; position: absolute; inset: 32px; border: 1px solid #1d3342; border-radius: 24px; pointer-events: none }
    .brand { display: flex; align-items: center; gap: 20px; color: #83f3d7; font-size: 25px; font-weight: 700; letter-spacing: .01em }
    .mark { display: grid; place-items: center; width: 58px; height: 58px; border: 1px solid #1ca88c; border-radius: 14px; background: #09261f; font-size: 28px }
    h1 { width: 900px; margin: 72px 0 24px; font-size: 66px; line-height: 1.08; letter-spacing: -.045em }
    p { width: 820px; margin: 0; color: #9eb1c4; font-size: 27px; line-height: 1.45 }
    strong { color: #55dfbd; font-weight: 600 }
    .signal { position: absolute; right: 100px; bottom: 88px; display: flex; align-items: center; gap: 12px; color: #9eb1c4; font: 17px ui-monospace, monospace }
    .signal i { width: 10px; height: 10px; border-radius: 50%; background: #35d6ad; box-shadow: 0 0 20px #35d6ad }
  </style></head><body><main><div class="brand"><span class="mark">⌕</span>PGSentinel</div><h1>PostgreSQL monitoring that explains problems.</h1><p>See <strong>what is wrong</strong>, why it matters, the evidence behind it, and what to investigate next.</p><div class="signal"><i></i>operations inbox for PostgreSQL</div></main></body></html>`);
  await page.screenshot({
    path: path.join(outputDir, "pgsentinel-social-preview.png"),
  });
  await browser.close();
  if (browserErrors.length)
    throw new Error(`Browser errors:\n${browserErrors.join("\n")}`);
} finally {
  vite.kill("SIGTERM");
}
