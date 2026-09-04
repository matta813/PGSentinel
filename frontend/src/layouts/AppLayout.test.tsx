import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { AppLayout } from "./AppLayout";
import { MonitoringProvider } from "../context/MonitoringContext";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

test("distinguishes a failed server list from a successful empty estate and retries", async () => {
  const store = new Map<string, string>();
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => store.set(key, value),
    removeItem: (key: string) => store.delete(key),
  });
  let serverAttempts = 0;
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    if (url.endsWith("/servers")) {
      serverAttempts += 1;
      if (serverAttempts === 1)
        return new Response(JSON.stringify({ error: "Unavailable" }), {
          status: 503,
        });
      return new Response(JSON.stringify([]), { status: 200 });
    }
    return new Response(JSON.stringify({ version: "0.7.0", commit: "test" }), {
      status: 200,
    });
  });
  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Routes>
          <Route
            element={
              <AppLayout
                username="admin"
                role="administrator"
                onLogout={() => undefined}
              />
            }
          >
            <Route path="queries" element={<p>Queries</p>} />
          </Route>
        </Routes>
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(
    await screen.findByText("Server list unavailable"),
  ).toBeInTheDocument();
  expect(screen.queryByText("No server configured")).not.toBeInTheDocument();
  expect(
    screen.getByRole("combobox", { name: "Global server" }),
  ).toBeDisabled();
  fireEvent.click(screen.getByRole("button", { name: "Retry server list" }));
  expect(await screen.findByText("No server configured")).toBeInTheDocument();
  expect(screen.queryByText("Server list unavailable")).not.toBeInTheDocument();
});

test("loads build metadata without using a stale cached version", async () => {
  const store = new Map<string, string>();
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => store.set(key, value),
    removeItem: (key: string) => store.delete(key),
  });
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockImplementation(
      async (input) =>
        new Response(
          JSON.stringify(
            String(input).endsWith("/servers")
              ? []
              : { version: "0.7.0", commit: "aa3e50f3" },
          ),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
    );

  render(
    <MemoryRouter>
      <MonitoringProvider>
        <Routes>
          <Route
            element={
              <AppLayout
                username="admin"
                role="administrator"
                onLogout={() => undefined}
              />
            }
          >
            <Route index element={<p>Dashboard</p>} />
          </Route>
        </Routes>
      </MonitoringProvider>
    </MemoryRouter>,
  );

  expect(await screen.findByText("v0.7.0")).toBeInTheDocument();
  await waitFor(() =>
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/version",
      expect.objectContaining({
        cache: "no-store",
        headers: expect.objectContaining({ "Cache-Control": "no-cache" }),
      }),
    ),
  );
});

test("shows truthful route-aware context and keeps database selection accessible", async () => {
  const store = new Map<string, string>();
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => store.set(key, value),
    removeItem: (key: string) => store.delete(key),
  });
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    const body = url.endsWith("/servers")
      ? [
          {
            id: "server-1",
            name: "Primary with a long server name",
            status: "healthy",
          },
        ]
      : url.endsWith("/databases")
        ? { databases: [{ Name: "customer_reporting_database" }] }
        : { version: "0.7.0", commit: "test" };
    return new Response(JSON.stringify(body), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  });
  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Routes>
          <Route
            element={
              <AppLayout
                username="admin"
                role="administrator"
                onLogout={() => undefined}
              />
            }
          >
            <Route path="queries" element={<p>Queries</p>} />
          </Route>
        </Routes>
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(
    await screen.findByRole("combobox", { name: "Global server" }),
  ).toBeInTheDocument();
  expect(
    await screen.findByRole("combobox", { name: "Global database" }),
  ).toBeInTheDocument();
  expect(screen.getByText("Latest snapshot")).toBeInTheDocument();
  expect(
    screen.queryByRole("combobox", { name: "Time range" }),
  ).not.toBeInTheDocument();
  await waitFor(() =>
    expect(
      screen.getByRole("combobox", { name: "Global database" }),
    ).toBeEnabled(),
  );
  fireEvent.change(screen.getByRole("combobox", { name: "Global database" }), {
    target: { value: "customer_reporting_database" },
  });
  await waitFor(() =>
    expect(
      screen.getByRole("combobox", { name: "Global database" }),
    ).toHaveValue("customer_reporting_database"),
  );
  fireEvent.click(screen.getByRole("button", { name: "Switch to dark theme" }));
  expect(document.documentElement.dataset.theme).toBe("dark");
});

test("renders every known database as a visible selectable option", async () => {
  const store = new Map<string, string>();
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => store.set(key, value),
    removeItem: (key: string) => store.delete(key),
  });
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    const body = url.endsWith("/servers")
      ? [{ id: "server-1", name: "Primary", status: "degraded", tags: [] }]
      : url.endsWith("/databases")
        ? {
            databases: [
              { Name: "postgres" },
              { Name: "tracearr" },
              { Name: "analytics" },
            ],
          }
        : { version: "test", commit: "test" };
    return new Response(JSON.stringify(body), { status: 200 });
  });
  render(
    <MemoryRouter initialEntries={["/tables"]}>
      <MonitoringProvider>
        <Routes>
          <Route
            element={
              <AppLayout
                username="admin"
                role="administrator"
                onLogout={() => undefined}
              />
            }
          >
            <Route path="tables" element={<p>Tables</p>} />
          </Route>
        </Routes>
      </MonitoringProvider>
    </MemoryRouter>,
  );
  const selector = await screen.findByRole("combobox", {
    name: "Global database",
  });
  await waitFor(() => expect(selector).toBeEnabled());
  expect(screen.getAllByRole("option").map((option) => option.textContent)).toEqual([
    "Primary · degraded",
    "All databases",
    "postgres",
    "tracearr",
    "analytics",
  ]);
  fireEvent.change(selector, { target: { value: "tracearr" } });
  expect(selector).toHaveValue("tracearr");
});
