import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { IncidentsPage } from "./IncidentsPage";
import {
  MonitoringProvider,
  useMonitoring,
} from "../context/MonitoringContext";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

function ServerSwitch() {
  const monitoring = useMonitoring();
  const location = useLocation();
  return (
    <>
      <button onClick={() => monitoring.setSelectedServerId("server-b")}>
        Select server B
      </button>
      <output aria-label="Current URL">{location.search}</output>
    </>
  );
}

test("clears an open incident when the global server scope changes", async () => {
  const store = new Map([["monitoring.server", "server-a"]]);
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => store.set(key, value),
    removeItem: (key: string) => store.delete(key),
  });
  const incident = {
    id: "incident-a",
    serverId: "server-a",
    title: "Server A incident",
    summary: "Only belongs to server A.",
    rationale: [],
    findings: [],
    timeline: [],
    severity: "HIGH",
    status: "active",
    startedAt: "2026-08-25T10:00:00Z",
    updatedAt: "2026-08-25T10:05:00Z",
  };
  const requests: string[] = [];
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    requests.push(url);
    if (url.endsWith("/servers"))
      return new Response(
        JSON.stringify([
          { id: "server-a", name: "Server A" },
          { id: "server-b", name: "Server B" },
        ]),
        { status: 200 },
      );
    if (url.includes("/incidents?"))
      return new Response(
        JSON.stringify(url.includes("serverId=server-a") ? [incident] : []),
        { status: 200 },
      );
    if (url.endsWith("/incidents/incident-a"))
      return new Response(JSON.stringify(incident), { status: 200 });
    return new Response(JSON.stringify({}), { status: 404 });
  });
  render(
    <MemoryRouter initialEntries={["/incidents?status=active"]}>
      <MonitoringProvider>
        <ServerSwitch />
        <IncidentsPage />
      </MonitoringProvider>
    </MemoryRouter>,
  );
  fireEvent.click(
    await screen.findByRole("button", { name: /Server A incident/ }),
  );
  expect(await screen.findByText("Why these were grouped")).toBeInTheDocument();
  expect(screen.getByLabelText("Current URL")).toHaveTextContent(
    "id=incident-a",
  );
  fireEvent.click(screen.getByRole("button", { name: "Select server B" }));
  expect(
    await screen.findByText("No correlated incidents"),
  ).toBeInTheDocument();
  expect(screen.queryByText("Why these were grouped")).not.toBeInTheDocument();
  await waitFor(() => {
    expect(screen.getByLabelText("Current URL")).not.toHaveTextContent("id=");
    expect(screen.getByLabelText("Current URL")).toHaveTextContent(
      "serverId=server-b",
    );
    expect(
      requests.some(
        (url) =>
          url.includes("/incidents?") && url.includes("serverId=server-b"),
      ),
    ).toBe(true);
  });
});

test("explains correlation and renders a chronological evidence timeline", async () => {
  const incident = {
    id: "a1b2c3d4e5f6a7b8c9d0e1f2",
    serverId: "server",
    title: "Overlapping PostgreSQL operational findings",
    summary:
      "Two findings may be related; this grouping does not establish causation.",
    rationale: ["connection pressure and lock waits are operationally related"],
    severity: "CRITICAL",
    status: "active",
    startedAt: "2026-08-25T10:00:00Z",
    updatedAt: "2026-08-25T10:05:00Z",
  };
  const requests: string[] = [];
  vi.spyOn(globalThis, "fetch").mockImplementation((input) => {
    const path = String(input);
    requests.push(path);
    if (path.startsWith("/api/v1/incidents?"))
      return Promise.resolve(
        new Response(JSON.stringify([incident]), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    if (path === `/api/v1/incidents/${incident.id}`)
      return Promise.resolve(
        new Response(
          JSON.stringify({
            ...incident,
            findings: [
              {
                id: "finding",
                serverId: "server",
                severity: "HIGH",
                category: "Locks",
                title: "Blocked queries",
                status: "active",
                startedAt: incident.startedAt,
                updatedAt: incident.updatedAt,
              },
            ],
            timeline: [
              {
                at: incident.startedAt,
                type: "finding_started",
                findingId: "finding",
                title: "Blocked queries",
                detail: "PGSentinel first observed this finding.",
                severity: "HIGH",
              },
            ],
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      );
    if (path === "/api/v1/servers")
      return Promise.resolve(
        new Response(JSON.stringify([{ id: "server", name: "Primary" }]), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    return Promise.resolve(new Response("{}", { status: 404 }));
  });
  render(
    <MemoryRouter>
      <MonitoringProvider>
        <IncidentsPage />
      </MonitoringProvider>
    </MemoryRouter>,
  );
  expect(
    await screen.findByText("Correlation is not causation."),
  ).toBeInTheDocument();
  fireEvent.click(
    await screen.findByRole("button", {
      name: /Overlapping PostgreSQL operational findings/,
    }),
  );
  expect(await screen.findByText("Why these were grouped")).toBeInTheDocument();
  expect(
    screen.getByText(
      "connection pressure and lock waits are operationally related",
    ),
  ).toBeInTheDocument();
  expect(screen.getByText("Finding observed")).toBeInTheDocument();
  expect(screen.getByRole("link", { name: /Blocked queries/ })).toHaveAttribute(
    "href",
    "/problems?id=finding",
  );
  expect(
    requests.some(
      (path) =>
        path.includes("/incidents?") && path.includes("serverId=server"),
    ),
  ).toBe(true);
});
