import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ProblemsPage } from "./ProblemsPage";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

test("shows suppression without hiding finding evidence", async () => {
  window.HTMLElement.prototype.scrollIntoView = vi.fn();
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(
      JSON.stringify([
        {
          id: "finding",
          serverId: "server",
          severity: "HIGH",
          category: "Replication",
          title: "Replica lag",
          summary: "Replay is delayed.",
          cause: "Observed lag.",
          impact: "Reads may be stale.",
          evidence: [{ label: "Replay lag", value: "120 seconds" }],
          suggestions: [],
          confidence: "HIGH",
          status: "active",
          startedAt: "2026-08-25T00:00:00Z",
          updatedAt: "2026-08-25T00:00:00Z",
          suppressed: true,
          maintenance: true,
          suppressionReason: "Maintenance: Planned failover",
        },
      ]),
      { status: 200, headers: { "Content-Type": "application/json" } },
    ),
  );
  render(
    <MemoryRouter initialEntries={["/problems?id=finding"]}>
      <ProblemsPage />
    </MemoryRouter>,
  );
  expect(
    await screen.findByText("Maintenance window active."),
  ).toBeInTheDocument();
  expect(
    screen.getByText(/Evidence and lifecycle state are preserved/),
  ).toBeInTheDocument();
  expect(screen.getByText("120 seconds")).toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: /suppress/i }),
  ).not.toBeInTheDocument();
});
