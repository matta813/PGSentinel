import type { Server } from "../../types";

export type ServerFormValue = { name: string; host: string; port: number; user: string; password: string; sslMode: string; tags: string[] };

export const initialServerForm: ServerFormValue = { name: "", host: "", port: 5432, user: "pgsentinel", password: "", sslMode: "prefer", tags: [] };

export function serverFormValue(server: Server): ServerFormValue {
  return { name: server.name, host: server.host, port: server.port, user: server.user, password: "", sslMode: server.sslMode, tags: server.tags };
}
