import { describe, expect, it, vi } from "vitest";

import {
  contentAccessError,
  ContentRevisionStore,
  createLineDiff,
  extractMainFrameVisibleText,
  fixedPageDiffSnapshot,
  fixedPageHasher,
  invalidateContentRevision,
  mapExtractionError,
  sha256Hex,
  validateStableIdentity,
} from "../src/content";

describe("fixed main-frame text extraction", () => {
  it.each([10_000, 50_000])("returns the exact %d character boundary", (limit) => {
    const text = "x".repeat(limit);
    expect(extractMainFrameVisibleText({ body: { innerText: text } }, limit)).toEqual({
      text,
      truncated: false,
      originalCharacterCount: limit,
      returnedCharacterCount: limit,
    });
  });

  it("uses 10,000 characters by default and reports truncation", () => {
    const text = "x".repeat(10_001);
    const result = extractMainFrameVisibleText({ body: { innerText: text } });
    expect(result.text).toHaveLength(10_000);
    expect(result).toMatchObject({
      truncated: true,
      originalCharacterCount: 10_001,
      returnedCharacterCount: 10_000,
    });
  });

  it("rejects limits above 50,000", () => {
    expect(() =>
      extractMainFrameVisibleText({ body: { innerText: "text" } }, 50_001),
    ).toThrow();
  });

  it("does not inspect iframe, HTML, cookies, storage, selection, or form values", () => {
    const forbiddenRead = () => {
      throw new Error("forbidden field was read");
    };
    const documentLike = {
      body: { innerText: "Visible main-frame text", forms: [{ value: "secret" }] },
      get cookie() {
        return forbiddenRead();
      },
      get documentElement() {
        return forbiddenRead();
      },
      get defaultView() {
        return forbiddenRead();
      },
      get localStorage() {
        return forbiddenRead();
      },
      get sessionStorage() {
        return forbiddenRead();
      },
      getSelection: forbiddenRead,
      frames: [{ document: { body: { innerText: "iframe secret" } } }],
    };

    expect(extractMainFrameVisibleText(documentLike)).toMatchObject({
      text: "Visible main-frame text",
    });
  });
});

describe("content revision invalidation", () => {
  it.each(["navigation", "reload", "url-changed", "tab-removed"] as const)(
    "invalidates a revision on %s",
    (type) => {
      const store = new ContentRevisionStore(() => "revision-1");
      const identity = {
        tabId: 7,
        url: "https://example.com/a",
        documentId: "document-1",
      };
      const revision = store.issue(identity);
      expect(store.isValid(revision, identity)).toBe(true);

      invalidateContentRevision(store, { type, tabId: 7 });

      expect(store.isValid(revision, identity)).toBe(false);
    },
  );

  it("persists only opaque revision identity in session storage", async () => {
    const set = vi.fn().mockResolvedValue(undefined);
    const store = await ContentRevisionStore.load(
      { get: vi.fn().mockResolvedValue({}), set },
      () => "revision-1",
    );
    store.issue({
      tabId: 7,
      url: "https://example.com/a",
      documentId: "document-1",
    });
    expect(set).toHaveBeenCalledWith({
      contentRevisions: {
        "revision-1": {
          tabId: 7,
          url: "https://example.com/a",
          documentId: "document-1",
        },
      },
    });
    expect(JSON.stringify(set.mock.calls)).not.toContain("page text");
  });
});

describe("content identity", () => {
  const before = {
    tabId: 7,
    url: "https://example.com/a",
    documentId: "document-1",
  };

  it.each([
    [{ ...before, url: "https://example.com/b" }],
    [{ ...before, documentId: "document-2" }],
  ])("rejects changed URL or document identity", (after) => {
    expect(() => validateStableIdentity(before, after)).toThrow("CONTENT_STALE");
  });

  it("accepts an unchanged main-frame document", () => {
    expect(() => validateStableIdentity(before, { ...before })).not.toThrow();
  });
});

describe("content access errors", () => {
  it.each([
    [
      { url: "https://example.com", incognito: false, ignored: false, hasPermission: false },
      "CONTENT_PERMISSION_REQUIRED",
    ],
    [
      { url: "chrome://settings", incognito: false, ignored: false, hasPermission: true },
      "CONTENT_NOT_ACCESSIBLE",
    ],
    [
      {
        url: "https://chromewebstore.google.com/detail/example/id",
        incognito: false,
        ignored: false,
        hasPermission: true,
      },
      "CONTENT_NOT_ACCESSIBLE",
    ],
    [
      { url: "https://example.com", incognito: true, ignored: false, hasPermission: true },
      "CONTENT_NOT_ACCESSIBLE",
    ],
    [
      { url: "https://example.com", incognito: false, ignored: true, hasPermission: true },
      "CONTENT_NOT_ACCESSIBLE",
    ],
  ])("maps access state to %s", (input, expected) => {
    expect(contentAccessError(input)).toBe(expected);
  });

  it("maps script injection failure without exposing page data", () => {
    expect(mapExtractionError(new Error("page supplied detail"))).toEqual({
      code: "CONTENT_EXTRACTION_FAILED",
      message: "Visible text extraction failed",
    });
  });
});

describe("page content hashing", () => {
  it("uses the standard lowercase SHA-256 digest", async () => {
    await expect(sha256Hex("abc")).resolves.toBe(
      "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
    );
  });

  it("returns only a digest and character count from the injected hasher", async () => {
    vi.stubGlobal("document", { body: { innerText: "abc" } });
    vi.stubGlobal("location", { href: "https://example.com/page" });
    try {
      const result = await fixedPageHasher();
      expect(result).toEqual({
        url: "https://example.com/page",
        sha256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
        characterCount: 3,
      });
      expect(result).not.toHaveProperty("text");
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it("hashes the full text while bounding the transient diff snapshot", async () => {
    vi.stubGlobal("document", { body: { innerText: "abcdef" } });
    vi.stubGlobal("location", { href: "https://example.com/page" });
    try {
      const result = await fixedPageDiffSnapshot(3);
      expect(result).toMatchObject({
        text: "abc",
        characterCount: 6,
        returnedCharacterCount: 3,
        truncated: true,
      });
      expect(result.sha256).toBe(await sha256Hex("abcdef"));
    } finally {
      vi.unstubAllGlobals();
    }
  });
});

describe("changed-lines-only diff", () => {
  it("omits unchanged lines and returns line-numbered deletes and inserts", () => {
    const result = createLineDiff(
      "unchanged\nold value\nstill unchanged",
      "unchanged\nnew value\nstill unchanged",
    );
    expect(result).toMatchObject({
      minimal: true,
      truncated: false,
      originalChangeCount: 2,
      returnedChangeCount: 2,
    });
    expect(result.changes).toEqual([
      { kind: "delete", oldLine: 2, text: "old value" },
      { kind: "insert", newLine: 2, text: "new value" },
    ]);
    expect(JSON.stringify(result.changes)).not.toContain("unchanged");
  });

  it("returns an empty change set for identical content", () => {
    expect(createLineDiff("same\ntext", "same\ntext").changes).toEqual([]);
  });

  it("bounds returned changed text and reports truncation", () => {
    const result = createLineDiff("0123456789", "abcdefghij", 5);
    expect(result.truncated).toBe(true);
    expect(result.returnedCharacterCount).toBe(5);
    expect(result.changes).toEqual([
      { kind: "delete", oldLine: 1, text: "01234", textTruncated: true },
    ]);
  });

  it("caps change entries independently of text size", () => {
    const before = Array.from({ length: 2_001 }, (_, index) => `line-${index}`).join("\n");
    const result = createLineDiff(before, "", 50_000);
    expect(result.originalChangeCount).toBe(2_001);
    expect(result.returnedChangeCount).toBe(2_000);
    expect(result.truncated).toBe(true);
  });

  it("marks the bounded replacement fallback as non-minimal", () => {
    const before = Array.from({ length: 1_500 }, (_, index) => `old-${index}`).join("\n");
    const after = Array.from({ length: 1_500 }, (_, index) => `new-${index}`).join("\n");
    expect(createLineDiff(before, after, 50_000).minimal).toBe(false);
  });
});
