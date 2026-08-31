import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ServersPage } from "./ServersPage";
import {
  MonitoringProvider,
  useMonitoring,
} from "../context/MonitoringContext";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

function GlobalSelection() {
  const monitoring = useMonitoring();
  return (
    <output aria-label="Global selection">{monitoring.selectedServerId}</output>
  );
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/servers"]}>
      <MonitoringProvider>
        <GlobalSelection />
        <ServersPage />
      </MonitoringProvider>
    </MemoryRouter>,
  );
}

test("labels lastConnectedAt truthfully and keeps connection errors visible", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(
      JSON.stringify([
        {
          id: "server-1",
          name: "Primary",
          host: "db.internal",
          port: 5432,
          user: "monitor",
          sslMode: "require",
          status: "error",
          tags: [],
          lastConnectedAt: "2026-08-31T08:00:00Z",
          lastError: "Connection refused",
        },
      ]),
      { status: 200, headers: { "Content-Type": "application/json" } },
    ),
  );
  renderPage();
  expect(
    await screen.findByRole("columnheader", { name: "Last connection" }),
  ).toBeInTheDocument();
  expect(
    screen.queryByRole("columnheader", { name: "Last collection" }),
  ).not.toBeInTheDocument();
  expect(screen.getByText("Connection error")).toHaveAttribute(
    "title",
    "Connection refused",
  );
});

test("deleting the selected server refreshes global context and falls back immediately", async () => {
  let deleted = false;
  const store = new Map<string, string>();
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => store.set(key, value),
    removeItem: (key: string) => store.delete(key),
  });
  vi.stubGlobal(
    "confirm",
    vi.fn(() => true),
  );
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const url = String(input);
    const method = init?.method ?? "GET";
    if (url.endsWith("/servers/primary") && method === "DELETE") {
      deleted = true;
      return new Response(null, { status: 204 });
    }
    if (url.endsWith("/servers"))
      return new Response(
        JSON.stringify(
          deleted
            ? [server("secondary", "Secondary")]
            : [server("primary", "Primary"), server("secondary", "Secondary")],
        ),
        { status: 200 },
      );
    return new Response(JSON.stringify({}), { status: 200 });
  });
  renderPage();
  expect(await screen.findByText("Primary")).toBeInTheDocument();
  expect(screen.getByLabelText("Global selection")).toHaveTextContent(
    "primary",
  );
  fireEvent.click(screen.getByRole("button", { name: "Delete Primary" }));
  await waitFor(() =>
    expect(screen.queryByText("Primary")).not.toBeInTheDocument(),
  );
  expect(screen.getByLabelText("Global selection")).toHaveTextContent(
    "secondary",
  );
  expect(store.get("monitoring.server")).toBe("secondary");
});

test("adding the first server refreshes and selects it globally", async () => {
  let created = false;
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const url = String(input);
    const method = init?.method ?? "GET";
    if (url.endsWith("/servers") && method === "POST") {
      created = true;
      return new Response(JSON.stringify(server("primary", "Primary")), {
        status: 201,
      });
    }
    if (url.endsWith("/servers"))
      return new Response(
        JSON.stringify(created ? [server("primary", "Primary")] : []),
        { status: 200 },
      );
    return new Response(JSON.stringify({}), { status: 200 });
  });
  renderPage();
  expect(await screen.findByText("No servers yet")).toBeInTheDocument();
  fireEvent.click(screen.getAllByRole("button", { name: "Add server" })[0]);
  fireEvent.change(screen.getByLabelText("Name"), {
    target: { value: "Primary" },
  });
  fireEvent.change(screen.getByLabelText("Host"), {
    target: { value: "db.internal" },
  });
  fireEvent.change(screen.getByLabelText(/Password/), {
    target: { value: "secret" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Save server" }));
  expect(await screen.findByText("Primary")).toBeInTheDocument();
  await waitFor(() =>
    expect(screen.getByLabelText("Global selection")).toHaveTextContent(
      "primary",
    ),
  );
});

function server(id: string, name: string) {
  return {
    id,
    name,
    host: `${id}.internal`,
    port: 5432,
    user: "monitor",
    sslMode: "require",
    status: "healthy",
    tags: [],
  };
}
