import {
  Activity,
  AlertTriangle,
  ChevronsUpDown,
  Database,
  FileSearch,
  Gauge,
  GitBranch,
  KeyRound,
  ListTree,
  Lock,
  LogOut,
  Menu,
  Moon,
  RefreshCw,
  ScrollText,
  Search,
  Settings,
  Sun,
  Table2,
  Timer,
  Users,
  Waves,
  X,
} from "lucide-react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import { useEffect, useState } from "react";
import { api } from "../api/client";
import { useMonitoring } from "../context/MonitoringContext";
import { useApi } from "../hooks/useApi";

const groups = [
  {
    label: "Monitoring",
    items: [
      ["/", "Dashboard", Gauge],
      ["/problems", "Problems", AlertTriangle],
      ["/incidents", "Incidents", ListTree],
    ],
  },
  {
    label: "Performance",
    items: [
      ["/queries", "Query Performance", Search],
      ["/wait-events", "Wait Events", Timer],
      ["/tables", "Tables", Table2],
      ["/indexes", "Index Analysis", KeyRound],
      ["/vacuum", "Vacuum", Activity],
      ["/locks", "Locks", Lock],
      ["/replication", "Replication", GitBranch],
      ["/wal", "WAL & Archive", Waves],
    ],
  },
  { label: "Management", items: [["/servers", "Servers", Database]] },
  {
    label: "System",
    admin: true,
    items: [
      ["/audit", "Audit Log", ScrollText],
      ["/users", "Users", Users],
      ["/settings", "Settings", Settings],
    ],
  },
] as const;
const databaseRoutes = new Set([
  "/queries",
  "/wait-events",
  "/tables",
  "/indexes",
  "/vacuum",
  "/locks",
  "/problems",
]);
const serverRoutes = new Set([
  ...databaseRoutes,
  "/replication",
  "/wal",
  "/incidents",
]);

export function AppLayout({
  username,
  role,
  onLogout,
}: {
  username: string;
  role: "administrator" | "operator" | "viewer";
  onLogout: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [dark, setDark] = useState(
    () => localStorage.getItem("theme") === "dark",
  );
  const monitoring = useMonitoring();
  const { pathname } = useLocation();
  const showsServer = serverRoutes.has(pathname);
  const showsDatabase = databaseRoutes.has(pathname);
  const contextLabel =
    pathname === "/"
      ? "Estate overview"
      : pathname === "/servers" ||
          pathname === "/settings" ||
          pathname === "/users" ||
          pathname === "/audit"
        ? "Administration"
        : "Current evidence";
  const evidenceLabel =
    pathname === "/problems"
      ? "Current findings"
      : pathname === "/incidents"
        ? "Current incidents"
        : "Latest snapshot";
  const { data: build } = useApi(
    () =>
      api.get<{ version: string; commit: string }>("/version", {
        cache: "no-store",
        headers: { "Cache-Control": "no-cache" },
      }),
    [],
  );
  useEffect(() => {
    document.documentElement.dataset.theme = dark ? "dark" : "light";
    localStorage.setItem("theme", dark ? "dark" : "light");
  }, [dark]);
  async function logout() {
    await api.post("/auth/logout");
    onLogout();
  }
  return (
    <div className="app-shell">
      {open && (
        <button
          className="nav-scrim"
          aria-label="Close navigation"
          onClick={() => setOpen(false)}
        />
      )}
      <aside className={`sidebar ${open ? "open" : ""}`}>
        <div className="brand">
          <span className="brand-mark">
            <FileSearch />
          </span>
          <div>
            <strong>PGSentinel</strong>
            <small>Database Intelligence</small>
          </div>
          <button
            className="icon-button sidebar-close"
            aria-label="Close navigation"
            onClick={() => setOpen(false)}
          >
            <X />
          </button>
        </div>
        <nav aria-label="Primary navigation">
          {groups
            .filter((group) => !("admin" in group) || role === "administrator")
            .map((group) => (
              <div className="nav-group" key={group.label}>
                <p>{group.label}</p>
                {group.items.map(([to, label, Icon]) => (
                  <NavLink
                    key={to}
                    to={to}
                    end={to === "/"}
                    onClick={() => setOpen(false)}
                  >
                    <Icon />
                    <span>{label}</span>
                  </NavLink>
                ))}
              </div>
            ))}
        </nav>
        <div className="sidebar-footer">
          <div className="monitoring-state">
            <span className="live-dot" />
            <span>
              <strong>Collector online</strong>
              <small>Monitoring enabled</small>
            </span>
          </div>
          <div className="build-meta">
            <span>v{build?.version ?? "dev"}</span>
            {build?.commit !== "unknown" && build?.commit && (
              <code>{build.commit.slice(0, 8)}</code>
            )}
          </div>
        </div>
      </aside>
      <main className="app-main">
        <header className="topbar">
          <div className="topbar-leading">
            <button
              className="icon-button menu-button"
              aria-label="Open navigation"
              onClick={() => setOpen(true)}
            >
              <Menu />
            </button>
            <span className="scope-label">{contextLabel}</span>
          </div>
          {showsServer && (
            <div className="context-controls">
              <label
                className={`context-selector server-context ${monitoring.serversError ? "context-error" : ""}`}
              >
                <span
                  className={`server-dot ${monitoring.selectedServer?.status?.toLowerCase() ?? "unknown"}`}
                />
                <span className="context-copy">
                  <small>Server</small>
                  <strong>{serverContextLabel(monitoring)}</strong>
                </span>
                <select
                  aria-label="Global server"
                  value={monitoring.selectedServerId}
                  disabled={
                    monitoring.serversLoading ||
                    !monitoring.servers.length ||
                    Boolean(monitoring.serversError)
                  }
                  onChange={(event) =>
                    monitoring.setSelectedServerId(event.target.value)
                  }
                >
                  {monitoring.servers.map((server) => (
                    <option value={server.id} key={server.id}>
                      {server.name} · {server.status}
                    </option>
                  ))}
                </select>
                <ChevronsUpDown />
              </label>
              {monitoring.serversError && (
                <button
                  className="icon-button context-retry"
                  aria-label="Retry server list"
                  title="Retry server list"
                  onClick={() => void monitoring.reloadServers()}
                >
                  <RefreshCw />
                </button>
              )}
              {showsDatabase && (
                <>
                  <label
                    className={`context-selector database-context ${monitoring.databasesError ? "context-error" : ""}`}
                  >
                    <Database />
                    <span className="database-select-field">
                      <small>Database</small>
                      <select
                        aria-label="Global database"
                        value={monitoring.selectedDatabase}
                        disabled={
                          monitoring.serversLoading ||
                          Boolean(monitoring.serversError) ||
                          !monitoring.selectedServerId ||
                          (monitoring.databasesLoading &&
                            !monitoring.databases.length) ||
                          (Boolean(monitoring.databasesError) &&
                            !monitoring.databases.length) ||
                          !monitoring.databases.length
                        }
                        onChange={(event) =>
                          monitoring.setSelectedDatabase(event.target.value)
                        }
                      >
                        <option value="">All databases</option>
                        {monitoring.databases.map((database) => (
                          <option value={database.Name} key={database.Name}>
                            {database.Name}
                          </option>
                        ))}
                      </select>
                    </span>
                    <ChevronsUpDown />
                  </label>
                  {monitoring.databasesError && (
                    <button
                      className="icon-button context-retry"
                      aria-label="Retry database list"
                      title="Retry database list"
                      onClick={() => void monitoring.reloadDatabases()}
                    >
                      <RefreshCw />
                    </button>
                  )}
                </>
              )}
            </div>
          )}
          <div className="topbar-actions">
            {showsServer && (
              <span className="evidence-mode">{evidenceLabel}</span>
            )}
            <button
              className="icon-button"
              aria-label={`Switch to ${dark ? "light" : "dark"} theme`}
              onClick={() => setDark((value) => !value)}
            >
              {dark ? <Sun /> : <Moon />}
            </button>
            <span className="account-label">
              {username}
              <small>{role}</small>
            </span>
            <button
              className="icon-button"
              aria-label="Sign out"
              onClick={() => void logout()}
            >
              <LogOut />
            </button>
          </div>
        </header>
        <div className="page">
          <Outlet />
        </div>
      </main>
    </div>
  );
}

function serverContextLabel(monitoring: ReturnType<typeof useMonitoring>) {
  if (monitoring.serversLoading) return "Loading servers…";
  if (monitoring.serversError) return "Server list unavailable";
  return monitoring.selectedServer?.name || "No server configured";
}
