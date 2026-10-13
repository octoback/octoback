import { describe, expect, it } from "vitest";
import { buildRestorePathPreview, compactRestorePaths, normalizeRestorePath } from "./preview";

describe("restore path preview", () => {
  it("combines a target directory with multiple selected source paths", () => {
    expect(buildRestorePathPreview("/restore", ["/etc/nginx.conf", "/var/www"])).toEqual(["/restore/etc/nginx.conf", "/restore/var/www"]);
  });

  it("maps a source root to the selected target directory", () => {
    expect(buildRestorePathPreview("/restore", ["/"])).toEqual(["/restore"]);
  });

  it("normalizes repeated and trailing separators without producing double slashes", () => {
    expect(normalizeRestorePath(" //restore///nested/ ")).toBe("/restore/nested");
    expect(buildRestorePathPreview("//restore///", ["///etc//nginx.conf//"])).toEqual(["/restore/etc/nginx.conf"]);
  });

  it("returns no preview until both target and source paths are selected", () => {
    expect(buildRestorePathPreview("", ["/etc"])).toEqual([]);
    expect(buildRestorePathPreview("/restore", [])).toEqual([]);
  });

  it("keeps a selected directory and removes its selected descendants", () => {
    expect(compactRestorePaths(["/home/www/.jenkins", "/home/www/.jenkins/.lastStarted", "/home/www/.jenkins/config.xml"])).toEqual(["/home/www/.jenkins"]);
  });

  it("keeps sibling paths separate and does not confuse shared prefixes", () => {
    expect(compactRestorePaths(["/home/www/app", "/home/www/application/config.xml", "/home/www/app/logs/app.log"])).toEqual(["/home/www/app", "/home/www/application/config.xml"]);
  });

  it("normalizes and deduplicates paths before compacting them", () => {
    expect(compactRestorePaths(["//home///www/app/", "/home/www/app", "/home/www/app/config.xml"])).toEqual(["/home/www/app"]);
  });

  it("uses the snapshot root when it is selected", () => {
    expect(compactRestorePaths(["/home/www/app", "/", "/etc/config"])).toEqual(["/"]);
  });
});
