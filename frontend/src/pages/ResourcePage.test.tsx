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

test.each([
  ["fresh", "Observed indexes", "0"],
  ["partial", "Index evidence unavailable", null],
  ["unavailable", "Index evidence unavailable", null],
] as const)(
  "distinguishes %s empty index evidence from an unavailable collection",
  async (state, expectedLabel, expectedValue) => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = String(input);
      if (url.endsWith("/servers"))
        return new Response(
          JSON.stringify([
            { id: "server-1", name: "Primary", status: "degraded", tags: [] },
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
              resource: "indexes",
              state,
              expectedIntervalSeconds: 300,
              consecutiveFailures: state === "fresh" ? 0 : 2,
            },
          ]),
          { status: 200 },
        );
      return new Response(JSON.stringify([]), { status: 200 });
    });
    render(
      <MemoryRouter initialEntries={["/indexes"]}>
        <MonitoringProvider>
          <Routes>
            <Route path="/:resource" element={<ResourcePage />} />
          </Routes>
        </MonitoringProvider>
      </MemoryRouter>,
    );
    expect(await screen.findByText(expectedLabel)).toBeInTheDocument();
    if (expectedValue !== null)
      expect(screen.getAllByText(expectedValue).length).toBeGreaterThan(0);
    else expect(screen.queryByText("Observed indexes")).not.toBeInTheDocument();
  },
);
