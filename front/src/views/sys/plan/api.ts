import { request } from "/src/api/service";
export interface BackupPlan {
  id: number;
  name: string;
  sourceId: number;
  repositoryId: number;
  repoTag?: string;
  keepLast?: number;
  keepHourly?: number;
  keepDaily?: number;
  keepWeekly?: number;
  keepMonthly?: number;
  keepYearly?: number;
  compression?: "auto" | "off" | "max";
  keepPolicy?: { type?: "count" | "time" | "none"; value?: Record<string, number | string> };
  retentionPolicy?: {
    mode?: "count" | "time" | "none";
    keepLast?: number;
    keepHourly?: number;
    keepDaily?: number;
    keepWeekly?: number;
    keepMonthly?: number;
    keepYearly?: number;
  };
  schedule: string;
  enabled: boolean;
  lastStatus?: string;
  lastError?: string;
  lastRunAt?: number;
}
export const getBackupPlanPage = (data: Record<string, unknown> = {}) => request({ url: "/plan/page", method: "post", data });
export const getBackupPlanList = () => request({ url: "/plan/list", method: "post", data: {} });
export const getBackupPlanInfo = (id: number) => request({ url: `/plan/info?id=${id}`, method: "post", data: {} });
export const createBackupPlan = (data: Partial<BackupPlan>) => request({ url: "/plan/create", method: "post", data });
export const updateBackupPlan = (id: number, data: Partial<BackupPlan>) => request({ url: `/plan/update?id=${id}`, method: "post", data });
export const deleteBackupPlan = (id: number) => request({ url: `/plan/delete?id=${id}`, method: "post", data: {} });
export const runBackupPlan = (id: number) => request({ url: `/plan/run?id=${id}`, method: "post", data: {} });
export const getBackupRunInfo = (id: number) => request({ url: `/plan/run/info?id=${id}`, method: "post", data: {} });
export interface BackupRunLogDetail {
  id: number;
  logs?: Array<{ id: number; level: string; message: string; createdAt: number }>;
  summary?: Record<string, unknown> | null;
}
export const getBackupRunLog = (id: number) => request({ url: `/plan/run/log?id=${id}`, method: "post", data: {} });
export const getBackupRunPage = (data: Record<string, unknown> = {}) => request({ url: "/plan/run/page", method: "post", data });
export interface BackupTask {
  id: number;
  planId: number;
  planName?: string;
  repositoryId?: number;
  status: string;
  progress: number;
  stage?: string;
  result?: string;
  error?: string;
  startedAt?: number;
  finishedAt?: number;
}
