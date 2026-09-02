import { useState } from "react";
import { api, APIError } from "../api/client";
import { useApi } from "../hooks/useApi";
import type { Server } from "../types";
import type { FindingSuppression, MaintenanceWindow, ThresholdResponse } from "../types/operator-controls";
import { ErrorState, Loading } from "./Status";
import { Notice } from "./UI";
import { MaintenanceWindowsSettings } from "./operator-controls/MaintenanceWindowsSettings";
import { SuppressionsSettings } from "./operator-controls/SuppressionsSettings";
import { ThresholdOverridesSettings } from "./operator-controls/ThresholdOverridesSettings";

export type OperatorControlAction = (task: () => Promise<unknown>, success: string, reload: () => unknown) => void;

export function OperatorControlsSettings() {
  const windows = useApi(() => api.get<MaintenanceWindow[]>("/maintenance-windows"), []);
  const suppressions = useApi(() => api.get<FindingSuppression[]>("/suppressions"), []);
  const thresholds = useApi(() => api.get<ThresholdResponse>("/threshold-overrides"), []);
  const servers = useApi(() => api.get<Server[]>("/servers"), []);
  const [message, setMessage] = useState("");
  const failed = windows.error || suppressions.error || thresholds.error || servers.error;
  if (windows.loading || suppressions.loading || thresholds.loading || servers.loading) return <Loading />;
  if (failed) return <ErrorState error={failed} retry={() => { void windows.reload(); void suppressions.reload(); void thresholds.reload(); void servers.reload(); }} />;
  const act: OperatorControlAction = (task, success, reload) => {
    setMessage("Saving operator control…");
    void task().then(() => { setMessage(success); reload(); }).catch((reason: unknown) => {
      setMessage(reason instanceof APIError ? `${reason.message}: ${reason.detail}` : "Unable to save operator control");
    });
  };
  return <div className="operator-controls">
    <MaintenanceWindowsSettings items={windows.data ?? []} servers={servers.data ?? []} reload={windows.reload} act={act} />
    <SuppressionsSettings items={suppressions.data ?? []} servers={servers.data ?? []} reload={suppressions.reload} act={act} />
    <ThresholdOverridesSettings response={thresholds.data} servers={servers.data ?? []} reload={thresholds.reload} act={act} />
    {message && <Notice>{message}</Notice>}
  </div>;
}
