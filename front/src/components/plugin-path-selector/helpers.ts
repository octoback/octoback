export interface PluginPathNode {
  title: string;
  key: string;
  children?: PluginPathNode[];
  isLeaf?: boolean;
  fileCount?: number;
  directoryCount?: number;
}

export function parsePathInput(value: string | string[] | null | undefined): string[] {
  const values = Array.isArray(value) ? value : String(value || "").split(/\r?\n/);
  return [...new Set(values.map((item) => String(item).trim()).filter(Boolean))];
}

export function getConfirmedPathSelection(multiple: boolean, selectedKeys: string[], checkedKeys: string[] | { checked: string[]; halfChecked: string[] }): string[] {
  if (!multiple) {
    return selectedKeys.slice(0, 1);
  }
  if (Array.isArray(checkedKeys)) {
    return parsePathInput(checkedKeys);
  }
  return parsePathInput(checkedKeys.checked);
}

export function findPathNode(nodes: PluginPathNode[], key: string): PluginPathNode | undefined {
  for (const node of nodes) {
    if (node.key === key) {
      return node;
    }
    if (node.children) {
      const foundNode = findPathNode(node.children, key);
      if (foundNode) {
        return foundNode;
      }
    }
  }
  return undefined;
}

export function isEmptyPathNode(node: PluginPathNode | undefined): boolean {
  return Boolean(node && node.fileCount === 0 && node.directoryCount === 0);
}

export function formatPathInput(value: string | string[] | null | undefined): string {
  return parsePathInput(value).join("\n");
}

export function getDirectoryRequestPath(path: unknown, cursor: unknown): string {
  const requestedPath = String(path ?? "").trim();
  if (requestedPath) {
    return requestedPath;
  }
  const requestedCursor = String(cursor ?? "").trim();
  if (requestedCursor) {
    return requestedCursor;
  }
  return "/";
}

export function updatePathNodeChildren(nodes: PluginPathNode[], key: string, children: PluginPathNode[]): boolean {
  for (const currentNode of nodes) {
    if (currentNode.key === key) {
      currentNode.children = children;
      currentNode.isLeaf = children.length === 0;
      return true;
    }
    if (currentNode.children && updatePathNodeChildren(currentNode.children, key, children)) {
      return true;
    }
  }
  return false;
}

export function updatePathNodeCounts(nodes: PluginPathNode[], key: string, result: unknown): boolean {
  for (const currentNode of nodes) {
    if (currentNode.key === key) {
      const data = result as any;
      const fileCount = Number(data?.fileCount);
      const directoryCount = Number(data?.directoryCount);
      if (!Number.isFinite(fileCount) || !Number.isFinite(directoryCount)) return false;
      currentNode.fileCount = fileCount;
      currentNode.directoryCount = directoryCount;
      return true;
    }
    if (currentNode.children && updatePathNodeCounts(currentNode.children, key, result)) return true;
  }
  return false;
}

export function normalizePathNodes(value: unknown): PluginPathNode[] {
  const source = Array.isArray(value) ? value : (value as any)?.paths || (value as any)?.records || (value as any)?.data || [];
  if (!Array.isArray(source)) return [];
  return source
    .map((item: any) => {
      const path = String(item?.path ?? item?.key ?? item?.name ?? item ?? "");
      const children = normalizePathNodes(item?.children);
      const fileCount = Number(item?.fileCount);
      const directoryCount = Number(item?.directoryCount);
      const hasCounts = Number.isFinite(fileCount) && Number.isFinite(directoryCount);
      const title = String(item?.title ?? item?.name ?? path);
      return { title, key: path, isLeaf: item?.isLeaf ?? children.length === 0, ...(hasCounts ? { fileCount, directoryCount } : {}), ...(children.length ? { children } : {}) };
    })
    .filter((item) => item.key);
}
