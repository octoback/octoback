import { request } from "/src/api/service";
export interface RestoreTask {
  id: number;
  title: string;
  repositoryId: number;
  repositoryName: string;
  snapshotId: string;
  sourceId?: number;
  targetAccessId: number;
  targetAccessName: string;
  targetPath: string;
  status: string;
  progress: number;
  stage?: string;
  error?: string;
  startedAt?: number;
  finishedAt?: number;
}
export const createRestoreTask = (data: { repositoryId: number; snapshotId: string; sourceId?: number; targetAccessId: number; targetPath: string; selectedPaths: string[] }) =>
  request({ url: "/restore/task/create", method: "post", data });
export const getRestoreTaskPage = (data: Record<string, unknown> = {}) => request({ url: "/restore/task/page", method: "post", data });
export const getRestoreTaskInfo = (id: number) => request({ url: `/restore/task/info?id=${id}`, method: "post", data: {} });
export const getRestoreTaskLog = (id: number) => request({ url: `/restore/task/log?id=${id}`, method: "post", data: {} });
export const deleteRestoreTask = (id: number) => request({ url: `/restore/task/delete?id=${id}`, method: "post", data: {} });
