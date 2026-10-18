import { request } from "/src/api/service";

export interface PluginField {
  key: string;
  title: string;
  type: string;
  required?: boolean;
  encrypt?: boolean;
  default?: unknown;
  placeholder?: string;
  helper?: string;
  mergeScript?: string;
  component?: Record<string, unknown>;
  options?: Array<{ label: string; value: string }>;
}
export interface PluginAction {
  name: string;
  title?: string;
  permission?: string;
}
export interface PluginMetadata {
  type: string;
  name: string;
  title: string;
  description: string;
  version: string;
  accessType?: string;
  icon?: string;
  group?: string;
  fields?: PluginField[];
  actions?: PluginAction[];
}
export interface PluginInstance {
  id: string | number;
  name: string;
  pluginType: string;
  pluginName: string;
  description?: string;
  config?: Record<string, unknown> & { configured?: boolean };
}
export interface PluginInstanceSimple {
  id: string | number;
  name: string;
  icon?: string;
  pluginName: string;
  pluginType: string;
}
export const getPluginMetadata = (data: { type?: string; name?: string } = {}): Promise<PluginMetadata[]> => request({ url: "/plugin/metadata", method: "post", data });
export const getPluginInstancePage = (data: { offset?: number; limit?: number; pluginType?: string; pluginName?: string; name?: string } = {}) => request({ url: "/plugin/instance/page", method: "post", data });
export const getPluginInstanceInfo = (id: string | number): Promise<PluginInstance> => request({ url: `/plugin/instance/info?id=${id}`, method: "post", data: {} });
export const getPluginInstanceSnapshots = (id: string | number) => request({ url: `/plugin/instance/snapshots?id=${id}`, method: "post", data: {} });
export const getPluginInstanceStats = (id: string | number) => request({ url: `/plugin/instance/stats?id=${id}`, method: "post", data: {} });
export const getPluginInstanceSnapshotStats = (id: string | number, snapshotId: string) =>
  request({ url: `/plugin/instance/snapshot-stats?id=${id}&snapshotId=${encodeURIComponent(snapshotId)}`, method: "post", data: {} });
export const refreshPluginInstanceRepositoryData = (id: string | number) => request({ url: `/plugin/instance/refresh-repository-data?id=${id}`, method: "post", data: {} });
export const checkPluginInstanceRepository = (id: string | number) => request({ url: `/plugin/instance/check?id=${id}`, method: "post", data: {} });
export const getPluginInstanceSnapshotBrowser = (id: string | number, snapshotId: string) =>
  request({ url: `/plugin/instance/snapshot-browser?id=${id}&snapshotId=${encodeURIComponent(snapshotId)}`, method: "post", data: {} });
export const forgetPluginInstanceSnapshot = (id: string | number, snapshotId: string) =>
  request({ url: `/plugin/instance/forget-snapshot?id=${id}&snapshotId=${encodeURIComponent(snapshotId)}`, method: "post", data: {} });
export const getPluginInstanceSimpleByIds = (ids: number[]): Promise<PluginInstanceSimple[]> => request({ url: "/plugin/instance/simpleByIds", method: "post", data: { ids } });
export const createPluginInstance = (data: { name: string; pluginType: string; pluginName: string; config: Record<string, unknown>; description?: string }) =>
  request({ url: "/plugin/instance/create", method: "post", data });
export const updatePluginInstance = (id: string | number, data: Partial<{ name: string; pluginType: string; pluginName: string; config: Record<string, unknown>; description: string }>) =>
  request({ url: `/plugin/instance/update?id=${id}`, method: "post", data });
export const deletePluginInstance = (id: string | number) => request({ url: `/plugin/instance/delete?id=${id}`, method: "post", data: {} });
export const executePluginInstanceAction = (id: string | number, action: string, params: Record<string, unknown> = {}) => request({ url: "/plugin/instance/action", method: "post", data: { id, action, params } });
