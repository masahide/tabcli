import { describe, expect, it } from "vitest";

import {
  normalizeTabIgnoreList,
  originPermissionPattern,
} from "../src/settings";

describe("settings", () => {
  it("normalizes an HTTP or HTTPS URL to its origin pattern", () => {
    expect(originPermissionPattern("https://example.com/path")).toBe("https://example.com/*");
    expect(originPermissionPattern("http://localhost:3000/path")).toBe("http://localhost/*");
    expect(() => originPermissionPattern("file:///tmp/example.html")).toThrow(
      "Only HTTP and HTTPS origins are supported",
    );
  });

  it("normalizes tabignore lines", () => {
    expect(normalizeTabIgnoreList("example.com\n*.example.org\n")).toEqual([
      "*://example.com/*",
      "*://*.example.org/*",
    ]);
  });
});
