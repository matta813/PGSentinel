import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { api } from "../api/client";
import { useApi } from "../hooks/useApi";
import type { DatabaseStat, Server, Snapshot } from "../types";
import { useLocation } from "react-router-dom";

export type TimeRange = "1h" | "6h" | "24h" | "7d" | "30d";

interface MonitoringContextValue {
  servers: Server[];
  serversLoading: boolean;
  serversError?: Error;
  reloadServers: () => Promise<void>;
  selectedServerId: string;
  setSelectedServerId: (id: string) => void;
  selectedServer?: Server;
  databases: DatabaseStat[];
  databasesLoading: boolean;
  databasesError?: Error;
  reloadDatabases: () => Promise<void>;
  selectedDatabase: string;
  setSelectedDatabase: (database: string) => void;
  timeRange: TimeRange;
  setTimeRange: (range: TimeRange) => void;
}

const MonitoringContext = createContext<MonitoringContextValue | null>(null);
const safeRange = (value: string | null): TimeRange =>
  ["1h", "6h", "24h", "7d", "30d"].includes(value ?? "")
    ? (value as TimeRange)
    : "24h";
const storage = () => globalThis.localStorage;
const databaseRoutes = new Set([
  "/queries",
  "/wait-events",
  "/tables",
  "/indexes",
  "/vacuum",
  "/locks",
  "/problems",
]);

export function MonitoringProvider({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();
  const databaseScopeEnabled = databaseRoutes.has(pathname);
  const serverResult = useApi(() => api.get<Server[]>("/servers"), []);
  const [selectedServerId, setSelectedServerIdState] = useState(
    () => storage()?.getItem("monitoring.server") ?? "",
  );
  const [databaseSelection, setDatabaseSelection] = useState(() => {
    const serverId = storage()?.getItem("monitoring.server") ?? "";
    return {
      serverId,
      database:
        storage()?.getItem(`monitoring.database.${serverId}`) ??
        storage()?.getItem("monitoring.database") ??
        "",
    };
  });
  const [timeRange, setTimeRangeState] = useState<TimeRange>(() =>
    safeRange(storage()?.getItem("monitoring.range") ?? null),
  );
  const effectiveServerId = serverResult.data?.some(
    (server) => server.id === selectedServerId,
  )
    ? selectedServerId
    : (serverResult.data?.[0]?.id ?? "");
  const databaseResult = useApi(
    () =>
      databaseScopeEnabled && effectiveServerId
        ? api.get<Snapshot>(`/servers/${effectiveServerId}/databases`)
        : Promise.resolve({ databases: [] } as unknown as Snapshot),
    [effectiveServerId, databaseScopeEnabled],
  );
  const databases = useMemo(
    () => {
      const seen = new Set<string>();
      return (databaseResult.data?.databases ?? []).reduce<DatabaseStat[]>(
        (items, database) => {
          const name =
            typeof database.Name === "string" ? database.Name.trim() : "";
          if (!name || seen.has(name)) return items;
          seen.add(name);
          items.push({ ...database, Name: name });
          return items;
        },
        [],
      );
    },
    [databaseResult.data],
  );
  const requestedDatabase =
    databaseSelection.serverId === effectiveServerId
      ? databaseSelection.database
      : effectiveServerId
        ? (storage()?.getItem(`monitoring.database.${effectiveServerId}`) ?? "")
        : "";
  const effectiveDatabase = databases.some(
    (database) => database.Name === requestedDatabase,
  )
    ? requestedDatabase
    : "";

  useEffect(() => {
    if (serverResult.loading || serverResult.error) return;
    if (effectiveServerId) {
      storage()?.setItem("monitoring.server", effectiveServerId);
    } else {
      storage()?.removeItem("monitoring.server");
    }
  }, [effectiveServerId, serverResult.error, serverResult.loading]);

  const value = useMemo<MonitoringContextValue>(
    () => ({
      servers: serverResult.data ?? [],
      serversLoading: serverResult.loading,
      serversError: serverResult.error,
      reloadServers: serverResult.reload,
      selectedServerId: effectiveServerId,
      setSelectedServerId: (id) => {
        setSelectedServerIdState(id);
        setDatabaseSelection({
          serverId: id,
          database: storage()?.getItem(`monitoring.database.${id}`) ?? "",
        });
        storage()?.setItem("monitoring.server", id);
      },
      selectedServer: serverResult.data?.find(
        (server) => server.id === effectiveServerId,
      ),
      databases,
      databasesLoading: databaseResult.loading,
      databasesError: databaseResult.error,
      reloadDatabases: databaseResult.reload,
      selectedDatabase: effectiveDatabase,
      setSelectedDatabase: (database) => {
        setDatabaseSelection({ serverId: effectiveServerId, database });
        if (database)
          storage()?.setItem(
            `monitoring.database.${effectiveServerId}`,
            database,
          );
        else storage()?.removeItem(`monitoring.database.${effectiveServerId}`);
        storage()?.removeItem("monitoring.database");
      },
      timeRange,
      setTimeRange: (range) => {
        setTimeRangeState(range);
        storage()?.setItem("monitoring.range", range);
      },
    }),
    [
      serverResult.data,
      serverResult.loading,
      serverResult.error,
      serverResult.reload,
      effectiveServerId,
      databases,
      databaseResult.loading,
      databaseResult.error,
      databaseResult.reload,
      effectiveDatabase,
      timeRange,
    ],
  );
  return (
    <MonitoringContext.Provider value={value}>
      {children}
    </MonitoringContext.Provider>
  );
}

// Context and provider intentionally share a module so their public contract stays together.
// eslint-disable-next-line react-refresh/only-export-components
export function useMonitoring() {
  const value = useContext(MonitoringContext);
  if (!value)
    throw new Error("useMonitoring must be used within MonitoringProvider");
  return value;
}
