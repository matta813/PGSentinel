import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ProblemsPage } from "./ProblemsPage";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

test("shows regression windows samples and significance beside the conclusion", async () => {
  window.HTMLElement.prototype.scrollIntoView = vi.fn();
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(
      JSON.stringify([
        {
          id: "regression",
          serverId: "primary",
          database: "app",
          resource: "42",
          severity: "HIGH",
          category: "Queries",
          title: "Persistent query latency regression detected",
          summary: "Two consecutive intervals exceeded the baseline.",
          cause: "",
          impact: "Latency may increase.",
          confidence: "HIGH",
          status: "active",
          startedAt: "2026-08-25T10:00:00Z",
          updatedAt: "2026-08-25T10:10:00Z",
          evidence: [
            { label: "Current window", value: "10:09 to 10:10" },
            { label: "Baseline window", value: "10:00 to 10:07" },
            { label: "Baseline samples", value: "7 intervals" },
            { label: "Previous interval mean", value: "30.00 ms" },
            { label: "Current interval mean", value: "32.00 ms" },
            { label: "Absolute difference", value: "+22.00 ms" },
            { label: "Relative difference", value: "+220.0%" },
            {
              label: "Significance",
              value: "Above median + 3 MAD, persistent for 2 intervals",
            },
          ],
          suggestions: [
            {
              title: "Correlate the regression window",
              detail: "Compare deployments and workload volume.",
            },
          ],
        },
      ]),
      { status: 200, headers: { "Content-Type": "application/json" } },
    ),
  );
  render(
    <MemoryRouter initialEntries={["/problems?id=regression"]}>
      <ProblemsPage />
    </MemoryRouter>,
  );
  expect(await screen.findByText("Current window")).toBeInTheDocument();
  for (const value of [
    "Baseline window",
    "7 intervals",
    "Previous interval mean",
    "+22.00 ms",
    "+220.0%",
    "Above median + 3 MAD, persistent for 2 intervals",
  ])
    expect(screen.getByText(value)).toBeInTheDocument();
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
