export function normalizeRestorePath(path: string): string {
  const segments = path.trim().split("/").filter(Boolean);
  if (!segments.length) {
    if (path.trim().startsWith("/")) {
      return "/";
    }
    return "";
  }
  return `/${segments.join("/")}`;
}

export function compactRestorePaths(paths: string[]): string[] {
  const normalizedPaths = Array.from(new Set(paths.map(normalizeRestorePath).filter(Boolean))).sort((firstPath, secondPath) => firstPath.length - secondPath.length);
  const compactedPaths: string[] = [];

  for (const candidatePath of normalizedPaths) {
    const coveredByParent = compactedPaths.some((parentPath) => {
      if (parentPath === "/") {
        return true;
      }
      return candidatePath === parentPath || candidatePath.startsWith(`${parentPath}/`);
    });
    if (!coveredByParent) {
      compactedPaths.push(candidatePath);
    }
  }

  return compactedPaths;
}

export function buildRestorePathPreview(targetPath: string, sourcePaths: string[]): string[] {
  const normalizedTarget = normalizeRestorePath(targetPath);
  if (!normalizedTarget || !sourcePaths.length) {
    return [];
  }

  return sourcePaths.map((sourcePath) => {
    const normalizedSource = normalizeRestorePath(sourcePath);
    if (normalizedSource === "/") {
      return normalizedTarget;
    }
    return `${normalizedTarget === "/" ? "" : normalizedTarget}${normalizedSource}` || "/";
  });
}
