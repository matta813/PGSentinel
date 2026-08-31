import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { IncidentsPage } from "./IncidentsPage";
import { MonitoringProvider } from "../context/MonitoringContext";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
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
