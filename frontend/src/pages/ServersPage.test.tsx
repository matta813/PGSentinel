import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, test, vi } from "vitest";
import { ServersPage } from "./ServersPage";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

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
  render(
    <MemoryRouter>
      <ServersPage />
    </MemoryRouter>,
  );
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
