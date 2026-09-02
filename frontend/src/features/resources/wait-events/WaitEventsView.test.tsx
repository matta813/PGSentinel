import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ResourcePage } from "../../../pages/ResourcePage";
import { MonitoringProvider } from "../../../context/MonitoringContext";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
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
