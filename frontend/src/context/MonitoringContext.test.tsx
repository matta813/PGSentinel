import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { MonitoringProvider, useMonitoring } from "./MonitoringContext";
import { MemoryRouter } from "react-router-dom";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

function Harness() {
  const context = useMonitoring();
  return (
    <>
      <span data-testid="selected-server">{context.selectedServer?.name}</span>
      <span data-testid="selected-database">
        {context.selectedDatabase || "all"}
      </span>
      <span data-testid="database-list">
        {context.databases.map((database) => database.Name).join(",")}
      </span>
      {context.databasesLoading && <span>loading databases</span>}
      {context.databasesError && <span>database unavailable</span>}
      {context.serversError && <span>server unavailable</span>}
      <button onClick={() => context.setSelectedServerId("secondary")}>
        Secondary
      </button>
      <button onClick={() => context.setSelectedDatabase("payments")}>
        Payments
      </button>
      <button onClick={() => context.setTimeRange("7d")}>Seven days</button>
      <button onClick={() => void context.reloadDatabases()}>
        Retry databases
      </button>
      <button onClick={() => void context.reloadServers()}>
        Retry servers
      </button>
    </>
  );
}

test("persists monitoring context and reloads databases when the server changes", async () => {
  const values = new Map<string, string>();
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
    removeItem: (key: string) => values.delete(key),
  });
  const requests: string[] = [];
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    requests.push(url);
    if (url.endsWith("/servers"))
      return new Response(
        JSON.stringify([
          { id: "primary", name: "Primary", status: "healthy" },
          { id: "secondary", name: "Secondary", status: "healthy" },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    return new Response(JSON.stringify({ databases: [{ Name: "payments" }] }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  });
  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Harness />
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(await screen.findByText("Primary")).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Secondary" }));
  await waitFor(() =>
    expect(screen.getByTestId("selected-server")).toHaveTextContent(
      "Secondary",
    ),
  );
  await waitFor(() =>
    expect(
      requests.some((url) => url.endsWith("/servers/secondary/databases")),
    ).toBe(true),
  );
  fireEvent.click(screen.getByRole("button", { name: "Payments" }));
  expect(screen.getByTestId("selected-database")).toHaveTextContent("payments");
  fireEvent.click(screen.getByRole("button", { name: "Seven days" }));
  expect(values.get("monitoring.range")).toBe("7d");
});

test("restores a persisted server and its database without leaking scope between servers", async () => {
  const values = new Map([
    ["monitoring.server", "secondary"],
    ["monitoring.database.secondary", "analytics"],
  ]);
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
    removeItem: (key: string) => values.delete(key),
  });
  vi.spyOn(globalThis, "fetch").mockImplementation(
    async (input) =>
      new Response(
        JSON.stringify(
          String(input).endsWith("/servers")
            ? [
                { id: "primary", name: "Primary" },
                { id: "secondary", name: "Secondary" },
              ]
            : { databases: [{ Name: "analytics" }] },
        ),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
  );
  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Harness />
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(await screen.findByTestId("selected-server")).toHaveTextContent(
    "Secondary",
  );
  await waitFor(() =>
    expect(screen.getByTestId("selected-database")).toHaveTextContent(
      "analytics",
    ),
  );
});

test("exposes database loading failures and can retry them", async () => {
  let databaseAttempts = 0;
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    if (String(input).endsWith("/servers"))
      return new Response(
        JSON.stringify([{ id: "primary", name: "Primary" }]),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    databaseAttempts += 1;
    return databaseAttempts === 1
      ? new Response(JSON.stringify({ error: "Unavailable" }), {
          status: 503,
          headers: { "Content-Type": "application/json" },
        })
      : new Response(JSON.stringify({ databases: [{ Name: "recovered" }] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
  });
  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Harness />
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(await screen.findByText("database unavailable")).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Retry databases" }));
  await waitFor(() =>
    expect(screen.getByTestId("database-list")).toHaveTextContent("recovered"),
  );
  expect(screen.queryByText("database unavailable")).not.toBeInTheDocument();
});

test("exposes server loading failures and recovers through reloadServers", async () => {
  let serverAttempts = 0;
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    if (!String(input).endsWith("/servers"))
      return new Response(JSON.stringify({ databases: [] }), { status: 200 });
    serverAttempts += 1;
    return serverAttempts === 1
      ? new Response(JSON.stringify({ error: "Unavailable" }), { status: 503 })
      : new Response(
          JSON.stringify([{ id: "primary", name: "Primary", tags: [] }]),
          { status: 200 },
        );
  });
  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Harness />
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(await screen.findByText("server unavailable")).toBeInTheDocument();
  expect(screen.getByTestId("selected-server")).toBeEmptyDOMElement();
  fireEvent.click(screen.getByRole("button", { name: "Retry servers" }));
  expect(await screen.findByText("Primary")).toBeInTheDocument();
  expect(screen.queryByText("server unavailable")).not.toBeInTheDocument();
});

test("removes a deleted persisted server and safely falls back without leaking its database", async () => {
  const values = new Map([
    ["monitoring.server", "primary"],
    ["monitoring.database.primary", "payments"],
    ["monitoring.database.secondary", "analytics"],
  ]);
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
    removeItem: (key: string) => values.delete(key),
  });
  let serverLoads = 0;
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    if (url.endsWith("/servers")) {
      serverLoads += 1;
      return new Response(
        JSON.stringify(
          serverLoads === 1
            ? [
                { id: "primary", name: "Primary", tags: [] },
                { id: "secondary", name: "Secondary", tags: [] },
              ]
            : [{ id: "secondary", name: "Secondary", tags: [] }],
        ),
        { status: 200 },
      );
    }
    return new Response(
      JSON.stringify({
        databases: [
          { Name: url.includes("secondary") ? "analytics" : "payments" },
        ],
      }),
      { status: 200 },
    );
  });
  render(
    <MemoryRouter initialEntries={["/queries"]}>
      <MonitoringProvider>
        <Harness />
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(await screen.findByText("Primary")).toBeInTheDocument();
  await waitFor(() =>
    expect(screen.getByTestId("selected-database")).toHaveTextContent(
      "payments",
    ),
  );
  fireEvent.click(screen.getByRole("button", { name: "Retry servers" }));
  expect(await screen.findByText("Secondary")).toBeInTheDocument();
  await waitFor(() => {
    expect(values.get("monitoring.server")).toBe("secondary");
    expect(screen.getByTestId("selected-database")).toHaveTextContent(
      "analytics",
    );
  });
});

test("keeps cached database metadata and selection after a failed reload", async () => {
  const values = new Map([
    ["monitoring.server", "primary"],
    ["monitoring.database.primary", "tracearr"],
  ]);
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
    removeItem: (key: string) => values.delete(key),
  });
  let databaseLoads = 0;
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    if (String(input).endsWith("/servers"))
      return new Response(
        JSON.stringify([{ id: "primary", name: "Primary", tags: [] }]),
        { status: 200 },
      );
    databaseLoads += 1;
    if (databaseLoads > 1)
      return new Response(JSON.stringify({ error: "Unavailable" }), {
        status: 503,
      });
    return new Response(
      JSON.stringify({
        databases: [
          { Name: "tracearr" },
          { Name: "" },
          { Name: "tracearr" },
          { Name: "postgres" },
        ],
      }),
      { status: 200 },
    );
  });
  render(
    <MemoryRouter initialEntries={["/tables"]}>
      <MonitoringProvider>
        <Harness />
      </MonitoringProvider>
    </MemoryRouter>,
  );
  await waitFor(() =>
    expect(screen.getByTestId("database-list")).toHaveTextContent(
      "tracearr,postgres",
    ),
  );
  expect(screen.getByTestId("selected-database")).toHaveTextContent("tracearr");
  fireEvent.click(screen.getByRole("button", { name: "Retry databases" }));
  expect(await screen.findByText("database unavailable")).toBeInTheDocument();
  expect(screen.getByTestId("database-list")).toHaveTextContent(
    "tracearr,postgres",
  );
  expect(screen.getByTestId("selected-database")).toHaveTextContent("tracearr");
});
