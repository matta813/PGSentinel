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
