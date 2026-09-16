import { describe, expect, it } from "vitest";
import { normalizeDetailsSearch, normalizeLegacyKind, sanitizeReturnTo } from "./-details-search";

describe("service details search", () => {
  it("extracts the return path from the malformed legacy compose URL", () => {
    expect(normalizeDetailsSearch({ kind: "compose?returnTo=%2Fprojects%2Fproject-1" })).toEqual({
      tab: undefined,
      returnTo: "/projects/project-1",
    });
  });

  it("does not validate legacy kind as the route identity", () => {
    expect(normalizeLegacyKind("compose?returnTo=%2Fprojects%2Fproject-1")).toBe("compose");
    expect(normalizeDetailsSearch({ kind: "database?returnTo=%2Fdatabases" })).toEqual({
      tab: undefined,
      returnTo: "/databases",
    });
  });

  it("rejects external return paths", () => {
    expect(sanitizeReturnTo("//evil.example/path")).toBeUndefined();
    expect(sanitizeReturnTo("https://evil.example/path")).toBeUndefined();
  });
});
