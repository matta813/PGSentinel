import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api } from '../api/client'
import { useApi } from '../hooks/useApi'
import type { DatabaseStat, Server, Snapshot } from '../types'

export type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d'

interface MonitoringContextValue {
  servers: Server[]
  serversLoading: boolean
  selectedServerId: string
  setSelectedServerId: (id: string) => void
  selectedServer?: Server
  databases: DatabaseStat[]
  selectedDatabase: string
  setSelectedDatabase: (database: string) => void
  timeRange: TimeRange
  setTimeRange: (range: TimeRange) => void
}

const MonitoringContext = createContext<MonitoringContextValue | null>(null)
const safeRange = (value: string | null): TimeRange => ['1h', '6h', '24h', '7d', '30d'].includes(value ?? '') ? value as TimeRange : '24h'
const storage = () => globalThis.localStorage

export function MonitoringProvider({ children }: { children: ReactNode }) {
  const servers = useApi(() => api.get<Server[]>('/servers'), [])
  const [selectedServerId, setSelectedServerIdState] = useState(() => storage()?.getItem('monitoring.server') ?? '')
  const [selectedDatabase, setSelectedDatabaseState] = useState(() => storage()?.getItem('monitoring.database') ?? '')
  const [timeRange, setTimeRangeState] = useState<TimeRange>(() => safeRange(storage()?.getItem('monitoring.range') ?? null))
  const effectiveServerId = servers.data?.some(server => server.id === selectedServerId) ? selectedServerId : servers.data?.[0]?.id ?? ''
  const databaseResult = useApi(() => effectiveServerId
    ? api.get<Snapshot>(`/servers/${effectiveServerId}/databases`).catch(() => ({ databases: [] } as unknown as Snapshot))
    : Promise.resolve({ databases: [] } as unknown as Snapshot), [effectiveServerId])
  const databases = useMemo(() => databaseResult.data?.databases ?? [], [databaseResult.data])
  const effectiveDatabase = databases.some(database => database.Name === selectedDatabase) ? selectedDatabase : ''

  useEffect(() => { if (effectiveServerId) storage()?.setItem('monitoring.server', effectiveServerId) }, [effectiveServerId])

  const value = useMemo<MonitoringContextValue>(() => ({
    servers: servers.data ?? [], serversLoading: servers.loading, selectedServerId: effectiveServerId,
    setSelectedServerId: id => { setSelectedServerIdState(id); setSelectedDatabaseState(''); storage()?.setItem('monitoring.server', id); storage()?.removeItem('monitoring.database') },
    selectedServer: servers.data?.find(server => server.id === effectiveServerId), databases, selectedDatabase: effectiveDatabase,
    setSelectedDatabase: database => { setSelectedDatabaseState(database); if (database) storage()?.setItem('monitoring.database', database); else storage()?.removeItem('monitoring.database') },
    timeRange, setTimeRange: range => { setTimeRangeState(range); storage()?.setItem('monitoring.range', range) },
  }), [servers.data, servers.loading, effectiveServerId, databases, effectiveDatabase, timeRange])
  return <MonitoringContext.Provider value={value}>{children}</MonitoringContext.Provider>
}

// Context and provider intentionally share a module so their public contract stays together.
// eslint-disable-next-line react-refresh/only-export-components
export function useMonitoring() {
  const value = useContext(MonitoringContext)
  if (!value) throw new Error('useMonitoring must be used within MonitoringProvider')
  return value
}
