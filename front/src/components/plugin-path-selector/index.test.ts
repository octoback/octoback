import { describe, expect, it } from "vitest";
import {
  findPathNode,
  formatPathInput,
  getConfirmedPathSelection,
  getDirectoryRequestPath,
  isEmptyPathNode,
  normalizePathNodes,
  parsePathInput,
  updatePathNodeChildren,
  updatePathNodeCounts
} from "/src/components/plugin-path-selector/helpers";

describe("plugin path selector helpers", () => {
  it("confirms only the selected tree node in single-select mode", () => {
    expect(getConfirmedPathSelection(false, ["/data/restore/test", "/ignored"], ["/checked"])).toEqual(["/data/restore/test"]);
  });

  it("confirms checked tree nodes in multiple-select mode", () => {
    expect(getConfirmedPathSelection(true, ["/selected"], { checked: ["/data", "/backup"], halfChecked: [] })).toEqual(["/data", "/backup"]);
  });

  it("finds a selected node and recognizes an empty directory", () => {
    const nodes = normalizePathNodes({
      paths: [
        {
          key: "/data",
          children: [
            { key: "/data/empty", fileCount: 0, directoryCount: 0 },
            { key: "/data/full", fileCount: 1, directoryCount: 0 }
          ]
        }
      ]
    });
    expect(isEmptyPathNode(findPathNode(nodes, "/data/empty"))).toBe(true);
    expect(isEmptyPathNode(findPathNode(nodes, "/data/full"))).toBe(false);
  });

  it("parses pasted paths into a unique array", () => {
    expect(parsePathInput("/etc\n /var/www\n/etc\n")).toEqual(["/etc", "/var/www"]);
  });

  it("formats array values for editing", () => {
    expect(formatPathInput(["/etc", "/var/www"])).toBe("/etc\n/var/www");
  });

  it("resolves a directory path for lazy loading", () => {
    expect(getDirectoryRequestPath(" /var ", undefined)).toBe("/var");
    expect(getDirectoryRequestPath(undefined, " /etc ")).toBe("/etc");
    expect(getDirectoryRequestPath(undefined, undefined)).toBe("/");
  });

  it("writes lazy-loaded children back into the tree", () => {
    const nodes = normalizePathNodes({ paths: [{ key: "/home", isLeaf: false }] });
    expect(updatePathNodeChildren(nodes, "/home", [{ key: "/home/www", title: "/home/www", isLeaf: true }])).toBe(true);
    expect(nodes[0].children?.[0].key).toBe("/home/www");
    expect(nodes[0].isLeaf).toBe(false);
  });

  it("marks a lazily loaded empty directory as a leaf so it will not remain expandable", () => {
    const nodes = normalizePathNodes({ paths: [{ key: "/empty", isLeaf: false }] });
    expect(updatePathNodeChildren(nodes, "/empty", [])).toBe(true);
    expect(nodes[0].children).toEqual([]);
    expect(nodes[0].isLeaf).toBe(true);
  });

  it("normalizes action directory results into tree nodes", () => {
    expect(
      normalizePathNodes({
        paths: [
          { path: "/etc", isLeaf: true },
          { name: "/var", children: [{ path: "/var/log" }] }
        ]
      })
    ).toEqual([
      { title: "/etc", key: "/etc", isLeaf: true },
      { title: "/var", key: "/var", isLeaf: false, children: [{ title: "/var/log", key: "/var/log", isLeaf: true }] }
    ]);
  });

  it("keeps path and merged item count as separate node fields", () => {
    const nodes = normalizePathNodes({ paths: [{ key: "/home", isLeaf: false }] });
    expect(updatePathNodeCounts(nodes, "/home", { fileCount: 4, directoryCount: 2 })).toBe(true);
    expect(nodes[0]).toEqual({ title: "/home", key: "/home", isLeaf: false, fileCount: 4, directoryCount: 2 });
    expect(nodes[0].fileCount! + nodes[0].directoryCount!).toBe(6);
  });
});
