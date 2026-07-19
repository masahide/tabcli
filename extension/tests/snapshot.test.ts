import { describe, expect, it } from "vitest";

import {
  buildSnapshot,
  normalizeTabIgnorePattern,
  type ChromeGroupLike,
  type ChromeTabLike,
} from "../src/snapshot";

describe("normalizeTabIgnorePattern", () => {
  it.each([
    ["example.com", "*://example.com/*"],
    ["*.example.com", "*://*.example.com/*"],
    ["https://example.com/*", "https://example.com/*"],
  ])("normalizes %s", (input, expected) => {
    expect(normalizeTabIgnorePattern(input)).toBe(expected);
  });

  it.each(["/example/", "!example.com", "example.*", "*example.com*"])(
    "rejects unsupported pattern %s",
    (input) => {
      expect(() => normalizeTabIgnorePattern(input)).toThrow();
    },
  );
});

describe("buildSnapshot", () => {
  const tabs: ChromeTabLike[] = [
    {
      id: 1,
      windowId: 10,
      index: 0,
      title: "Docs",
      url: "https://example.com/docs",
      active: true,
      pinned: false,
      groupId: 100,
      incognito: false,
      lastAccessed: 1_700_000_000_000.625,
    },
    {
      id: 2,
      windowId: 10,
      index: 1,
      title: "Settings",
      url: "chrome://settings/",
      active: false,
      pinned: false,
      groupId: -1,
      incognito: false,
    },
    {
      id: 3,
      windowId: 20,
      index: 0,
      title: "Private",
      url: "https://private.example/",
      active: true,
      pinned: false,
      groupId: -1,
      incognito: true,
    },
    {
      id: 4,
      windowId: 10,
      index: 2,
      title: "Ignored",
      url: "https://ignored.example/page",
      active: false,
      pinned: false,
      groupId: -1,
      incognito: false,
    },
  ];
  const groups: ChromeGroupLike[] = [
    { id: 100, windowId: 10, title: "Work", color: "blue", collapsed: false },
    { id: 200, windowId: 20, title: "Private", color: "red", collapsed: false },
  ];

  it("excludes incognito and tabignore targets and marks restricted URLs", () => {
    const snapshot = buildSnapshot(tabs, groups, ["ignored.example"]);

    expect(snapshot.tabs.map((tab) => tab.id)).toEqual([1, 2]);
    expect(snapshot.tabs[0]).toMatchObject({ operable: true, groupId: 100 });
	expect(snapshot.tabs[0].lastAccessed).toBe(1_700_000_000_000);
    expect(snapshot.tabs[1]).toMatchObject({ operable: false, groupId: -1 });
    expect(snapshot.groups).toEqual([
      expect.objectContaining({ id: 100, tabIds: [1] }),
    ]);
  });
});
