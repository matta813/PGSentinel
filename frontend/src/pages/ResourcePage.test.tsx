import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ResourcePage } from "./ResourcePage";
import { MonitoringProvider } from "../context/MonitoringContext";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

test("shows a retryable error instead of claiming no server is configured", async () => {
  let serverAttempts = 0;
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    if (url.endsWith("/servers")) {
      serverAttempts += 1;
      return serverAttempts === 1
        ? new Response(JSON.stringify({ error: "Unavailable" }), {
            status: 503,
          })
        : new Response(
            JSON.stringify([
              { id: "server-1", name: "Primary", status: "healthy", tags: [] },
            ]),
            { status: 200 },
          );
    }
    if (url.endsWith("/databases"))
      return new Response(JSON.stringify({ databases: [] }), { status: 200 });
    return new Response(JSON.stringify([]), { status: 200 });
  });
  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Routes>
          <Route path="/:resource" element={<ResourcePage />} />
        </Routes>
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(await screen.findByText("Unable to load data")).toBeInTheDocument();
  expect(screen.queryByText("No server configured")).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Try again" }));
  expect(await screen.findByText("Primary")).toBeInTheDocument();
});

test.each([
  ["fresh", true],
  ["unavailable", false],
] as const)(
  "%s empty lock evidence only makes a positive claim when fresh",
  async (state, expectsPositive) => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = String(input);
      if (url.endsWith("/servers"))
        return new Response(
          JSON.stringify([
            { id: "server-1", name: "Primary", status: "healthy", tags: [] },
          ]),
          { status: 200 },
        );
      if (url.endsWith("/databases"))
        return new Response(JSON.stringify({ databases: [] }), { status: 200 });
      if (url.endsWith("/freshness"))
        return new Response(
          JSON.stringify([
            {
              serverId: "server-1",
              resource: "locks",
              state,
              expectedIntervalSeconds: 30,
              consecutiveFailures: state === "fresh" ? 0 : 1,
              ageSeconds: state === "fresh" ? 5 : undefined,
            },
          ]),
          { status: 200 },
        );
      return new Response(JSON.stringify([]), { status: 200 });
    });
    render(
      <MemoryRouter initialEntries={["/locks"]}>
        <MonitoringProvider>
          <Routes>
            <Route path="/:resource" element={<ResourcePage />} />
          </Routes>
        </MonitoringProvider>
      </MemoryRouter>,
    );
    if (expectsPositive) {
      expect(
        await screen.findByText("No blocking sessions detected"),
      ).toBeInTheDocument();
      expect(
        screen.queryByText("No current lock evidence available"),
      ).not.toBeInTheDocument();
    } else {
      expect(
        await screen.findByText("No current lock evidence available"),
      ).toBeInTheDocument();
      expect(
        screen.queryByText("No blocking sessions detected"),
      ).not.toBeInTheDocument();
    }
  },
);

test("warns before rendering cached evidence as current", async () => {
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    if (url.endsWith("/servers"))
      return new Response(
        JSON.stringify([
          { id: "server-1", name: "Primary", status: "degraded", tags: [] },
        ]),
        { status: 200 },
      );
    if (url.endsWith("/queries"))
      return new Response(
        JSON.stringify([
          {
            QueryID: "1",
            Query: "select 1",
            Database: "app",
            Calls: 20,
            MeanExecMS: 3,
            TotalExecMS: 60,
            ImpactScore: 1,
          },
        ]),
        { status: 200 },
      );
    return new Response(
      JSON.stringify([
        {
          serverId: "server-1",
          resource: "queries",
          state: "unavailable",
          lastSuccessfulCollection: "2026-08-25T10:00:00Z",
          expectedIntervalSeconds: 30,
          consecutiveFailures: 2,
          errorSummary:
            "Collection failed; the last successful evidence is preserved.",
        },
      ]),
      { status: 200 },
    );
  });
  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Routes>
          <Route path="/:resource" element={<ResourcePage />} />
        </Routes>
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(await screen.findByText("select 1")).toBeInTheDocument();
  expect(
    screen.getByText("Current evidence is unavailable"),
  ).toBeInTheDocument();
  expect(
    screen.getByText(/last successful evidence is preserved/i),
  ).toBeInTheDocument();
});

test("renders resource data when optional freshness metadata is unavailable", async () => {
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    if (url.endsWith("/servers"))
      return new Response(
        JSON.stringify([
          { id: "server-1", name: "Primary", status: "healthy", tags: [] },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    if (url.endsWith("/queries"))
      return new Response(
        JSON.stringify([
          {
            QueryID: "1",
            Query: "select 1",
            Database: "app",
            Calls: 20,
            MeanExecMS: 3,
            TotalExecMS: 60,
            ImpactScore: 1,
          },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    return new Response(JSON.stringify({ error: "Resource not found" }), {
      status: 404,
      headers: { "Content-Type": "application/json" },
    });
  });

  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Routes>
          <Route path="/:resource" element={<ResourcePage />} />
        </Routes>
      </MonitoringProvider>
    </MemoryRouter>,
  );

  expect(await screen.findByText("select 1")).toBeInTheDocument();
  expect(screen.queryByText("Unable to load data")).not.toBeInTheDocument();
  expect(
    screen.getByText("Current evidence is unavailable"),
  ).toBeInTheDocument();
});

test("filters query text and sorts numerical workload columns", async () => {
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    if (url.endsWith("/servers"))
      return new Response(
        JSON.stringify([
          { id: "server-1", name: "Primary", status: "healthy", tags: [] },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    if (url.endsWith("/databases"))
      return new Response(JSON.stringify({ databases: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    if (url.endsWith("/queries"))
      return new Response(
        JSON.stringify([
          {
            QueryID: "1",
            Query: "select slow",
            Database: "app",
            Calls: 2,
            MeanExecMS: 30,
            TotalExecMS: 60,
            ImpactScore: 90,
          },
          {
            QueryID: "2",
            Query: "select fast",
            Database: "app",
            Calls: 100,
            MeanExecMS: 1,
            TotalExecMS: 100,
            ImpactScore: 10,
          },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    return new Response(JSON.stringify([]), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  });
  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Routes>
          <Route path="/:resource" element={<ResourcePage />} />
        </Routes>
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(await screen.findByText("select slow")).toBeInTheDocument();
  fireEvent.change(screen.getByRole("textbox", { name: "Search query text" }), {
    target: { value: "fast" },
  });
  expect(screen.queryByText("select slow")).not.toBeInTheDocument();
  expect(screen.getByText("select fast")).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Sort by Calls" }));
  expect(
    screen.getByRole("button", { name: "Sort by Calls" }),
  ).toBeInTheDocument();
});

test("distinguishes a database filter with no rows from missing collection data", async () => {
  const values = new Map([
    ["monitoring.server", "server-1"],
    ["monitoring.database.server-1", "analytics"],
  ]);
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
    removeItem: (key: string) => values.delete(key),
  });
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    const body = url.endsWith("/servers")
      ? [{ id: "server-1", name: "Primary", status: "healthy" }]
      : url.endsWith("/databases")
        ? { databases: [{ Name: "analytics" }, { Name: "app" }] }
        : url.endsWith("/tables")
          ? [
              {
                Database: "app",
                Schema: "public",
                Table: "orders",
                EstimatedRows: 10,
                TotalSize: 1024,
                DeadTuples: 0,
                LiveTuples: 10,
                IndexScans: 2,
              },
            ]
          : [];
    return new Response(JSON.stringify(body), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  });
  render(
    <MemoryRouter initialEntries={["/tables"]}>
      <MonitoringProvider>
        <Routes>
          <Route path="/:resource" element={<ResourcePage />} />
        </Routes>
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(
    await screen.findByText("No data for the selected database"),
  ).toBeInTheDocument();
  expect(screen.getByText(/none for analytics/i)).toBeInTheDocument();
  expect(screen.queryByText("No data collected yet")).not.toBeInTheDocument();
});

test("renders and filters current wait-event aggregation including unknown classes", async () => {
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    const body = url.endsWith("/servers")
      ? [{ id: "server-1", name: "Primary", status: "healthy", tags: [] }]
      : url.endsWith("/databases")
        ? { databases: [{ Name: "app" }, { Name: "audit" }] }
        : url.endsWith("/freshness")
          ? [
              {
                serverId: "server-1",
                resource: "wait-events",
                state: "fresh",
                ageSeconds: 4,
                expectedIntervalSeconds: 30,
                consecutiveFailures: 0,
              },
            ]
          : url.endsWith("/wait-events")
            ? [
                {
                  PID: 10,
                  Database: "app",
                  User: "api",
                  Application: "web",
                  State: "active",
                  WaitEventType: "Lock",
                  WaitEvent: "transactionid",
                  Query: "update orders",
                  QueryAgeSeconds: 47,
                  TransactionAgeSeconds: 52,
                  TransactionStartedAt: "2026-08-31T10:00:00Z",
                },
                {
                  PID: 11,
                  Database: "audit",
                  User: "worker",
                  Application: "jobs",
                  State: "active",
                  WaitEventType: "FutureClass",
                  WaitEvent: "FutureWait",
                  Query: "select audit",
                  QueryAgeSeconds: 9,
                  TransactionAgeSeconds: 0,
                },
              ]
            : [];
    return new Response(JSON.stringify(body), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  });
  render(
    <MemoryRouter initialEntries={["/wait-events"]}>
      <MonitoringProvider>
        <Routes>
          <Route path="/:resource" element={<ResourcePage />} />
        </Routes>
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(await screen.findByText("Waiting sessions")).toBeInTheDocument();
  expect(screen.getAllByText("FutureClass").length).toBeGreaterThan(0);
  expect(
    screen.getByLabelText("Wait class distribution").textContent?.match(/50\.0%/g),
  ).toHaveLength(2);
  fireEvent.change(
    screen.getByRole("combobox", { name: "Filter wait class" }),
    { target: { value: "Lock" } },
  );
  expect(screen.getByText("update orders")).toBeInTheDocument();
  expect(screen.queryByText("select audit")).not.toBeInTheDocument();
});

test.each([
  ["fresh", true],
  ["unavailable", false],
] as const)(
  "%s empty wait evidence only claims no waits when fresh",
  async (state, healthy) => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = String(input);
      const body = url.endsWith("/servers")
        ? [{ id: "server-1", name: "Primary", status: "healthy", tags: [] }]
        : url.endsWith("/databases")
          ? { databases: [] }
          : url.endsWith("/freshness")
            ? [
                {
                  serverId: "server-1",
                  resource: "wait-events",
                  state,
                  expectedIntervalSeconds: 30,
                  consecutiveFailures: state === "fresh" ? 0 : 1,
                },
              ]
            : [];
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });
    render(
      <MemoryRouter initialEntries={["/wait-events"]}>
        <MonitoringProvider>
          <Routes>
            <Route path="/:resource" element={<ResourcePage />} />
          </Routes>
        </MonitoringProvider>
      </MemoryRouter>,
    );
    if (healthy)
      expect(
        await screen.findByText(
          "No sessions are currently reporting a PostgreSQL wait event.",
        ),
      ).toBeInTheDocument();
    else {
      expect(
        await screen.findByText("Current wait state unavailable"),
      ).toBeInTheDocument();
      expect(
        screen.queryByText(
          "No sessions are currently reporting a PostgreSQL wait event.",
        ),
      ).not.toBeInTheDocument();
    }
  },
);
