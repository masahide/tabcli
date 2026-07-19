import { afterEach, describe, expect, it, vi } from "vitest";

import { compareTabContent, diffTabContent } from "../src/content-service";

const pages = new Map([
  [7, { url: "https://example.com/a", title: "A", text: "same\nold\ntail", sha256: "hash-a" }],
  [9, { url: "https://example.com/b", title: "B", text: "same\nnew\ntail", sha256: "hash-b" }],
]);

afterEach(() => {
  vi.unstubAllGlobals();
});

function installChromeFixture(hasPermission = true) {
  const executeScript = vi.fn(async (details: {
    target: { tabId: number };
    func: { name: string };
    args?: number[];
  }) => {
    const page = pages.get(details.target.tabId);
    if (page === undefined) throw new Error("missing tab");
    const documentId = `document-${details.target.tabId}`;
    if (details.func.name === "fixedIdentityProbe") {
      return [{ documentId, result: { url: page.url } }];
    }
    if (details.func.name === "fixedPageHasher") {
      return [{
        documentId,
        result: { url: page.url, sha256: page.sha256, characterCount: page.text.length },
      }];
    }
    if (details.func.name === "fixedPageDiffSnapshot") {
      const maxCharacters = details.args?.[0] ?? 50_000;
      return [{
        documentId,
        result: {
          url: page.url,
          sha256: page.sha256,
          characterCount: page.text.length,
          text: page.text.slice(0, maxCharacters),
          truncated: page.text.length > maxCharacters,
          returnedCharacterCount: Math.min(page.text.length, maxCharacters),
        },
      }];
    }
    throw new Error(`unexpected function: ${details.func.name}`);
  });
  vi.stubGlobal("chrome", {
    tabs: {
      get: vi.fn(async (tabId: number) => {
        const page = pages.get(tabId);
        if (page === undefined) throw new Error("missing tab");
        return { id: tabId, url: page.url, title: page.title, incognito: false };
      }),
    },
    permissions: { contains: vi.fn(async () => hasPermission) },
    scripting: { executeScript },
  });
  return executeScript;
}

describe("content comparison service", () => {
  it("returns hashes without returning visible page text", async () => {
    installChromeFixture();
    const result = await compareTabContent({ tabIds: [7, 9] }, []);
    expect(result).toMatchObject({
      hashAlgorithm: "SHA-256",
      match: false,
      tabs: [
        { tabId: 7, sha256: "hash-a" },
        { tabId: 9, sha256: "hash-b" },
      ],
    });
    const encoded = JSON.stringify(result);
    expect(encoded).not.toContain("same");
    expect(encoded).not.toContain("old");
    expect(encoded).not.toContain("new");
  });

  it("returns only changed lines for diff and omits source snapshots", async () => {
    installChromeFixture();
    const result = await diffTabContent(
      { tabIds: [7, 9], maxChars: 50_000, maxDiffChars: 20_000 },
      [],
    );
    expect(result.changes).toEqual([
      { kind: "delete", oldLine: 2, text: "old" },
      { kind: "insert", newLine: 2, text: "new" },
    ]);
    expect(result.untrustedContent).toBe(true);
    const encoded = JSON.stringify(result);
    expect(encoded).not.toContain("same");
    expect(encoded).not.toContain("tail");
  });

  it("honors Chrome site-access restrictions before injecting", async () => {
    const executeScript = installChromeFixture(false);
    await expect(compareTabContent({ tabIds: [7, 9] }, [])).rejects.toMatchObject({
      code: "CONTENT_PERMISSION_REQUIRED",
    });
    expect(executeScript).not.toHaveBeenCalled();
  });

  it("maps a disappeared tab without reporting a protocol mismatch", async () => {
    const executeScript = installChromeFixture();
    await expect(compareTabContent({ tabIds: [7, 11] }, [])).rejects.toMatchObject({
      code: "TAB_NOT_FOUND",
    });
    expect(executeScript).not.toHaveBeenCalled();
  });
});
