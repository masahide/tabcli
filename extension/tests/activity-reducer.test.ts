import { describe, expect, it } from "vitest";

import {
  activityForTab,
  createActivityState,
  reconcileActivity,
  reduceActivity,
} from "../src/activity-reducer";

describe("activity reducer events", () => {
  it("records creation, activation, updates, movement, attachment and grouping", () => {
    let state = createActivityState("session-1", 1_000);
    state = reduceActivity(state, { type: "tab-created", tabId: 7, incognito: false }, 1_100);
    expect(state.records[7]).toMatchObject({
      createdAt: 1_100,
      firstObservedAt: 1_100,
      activationCount: 0,
      activityDataCompleteness: "created_observed",
    });

    state = reduceActivity(state, { type: "tab-activated", tabId: 7 }, 1_200);
    expect(state.records[7].activationCount).toBe(1);

    state = reduceActivity(state, { type: "tab-updated", tabId: 7, groupChanged: true }, 1_300);
    expect(state.records[7].lastGroupChangedAt).toBe(1_300);

    state = reduceActivity(state, { type: "tab-moved", tabId: 7 }, 1_400);
    state = reduceActivity(state, { type: "tab-detached", tabId: 7 }, 1_500);
    state = reduceActivity(state, { type: "tab-attached", tabId: 7 }, 1_600);
    expect(state.records[7].lastMovedAt).toBe(1_600);

    state = reduceActivity(state, { type: "group-changed", tabIds: [7] }, 1_700);
    expect(state.records[7].lastGroupChangedAt).toBe(1_700);
  });

  it("deletes metadata when a tab is removed", () => {
    let state = createActivityState("session-1", 1_000);
    state = reduceActivity(state, { type: "tab-created", tabId: 7, incognito: false }, 1_100);
    state = reduceActivity(state, { type: "tab-removed", tabId: 7 }, 1_200);
    expect(state.records[7]).toBeUndefined();
  });

  it("never records incognito tabs", () => {
    let state = createActivityState("session-1", 1_000);
    state = reduceActivity(state, { type: "tab-created", tabId: 8, incognito: true }, 1_100);
    state = reconcileActivity(
      state,
      [
        { id: 8, incognito: true },
        { id: 9, incognito: false },
      ],
      "session-1",
      1_200,
    );
    expect(state.records[8]).toBeUndefined();
    expect(state.records[9]).toBeDefined();
    state = reduceActivity(state, { type: "tab-removed", tabId: 9 }, 1_300);
    expect(state.records[9]).toBeUndefined();
  });
});

describe("activity completeness", () => {
  it("does not invent creation time for tabs that predate tracking", () => {
    let state = createActivityState("session-1", 1_000);
    state = reconcileActivity(
      state,
      [{ id: 1, incognito: false }],
      "session-1",
      1_100,
    );
    expect(state.records[1]).toMatchObject({
      createdAt: null,
      firstObservedAt: 1_100,
      trackingSince: 1_000,
      activityDataCompleteness: "tracking_started_after_creation",
    });

    state = reduceActivity(state, { type: "tab-created", tabId: 2, incognito: false }, 1_200);
    expect(state.records[2]).toMatchObject({
      createdAt: 1_200,
      activityDataCompleteness: "created_observed",
    });
  });

  it("returns snapshot-only completeness when metadata is absent", () => {
    const state = createActivityState("session-1", 1_000);
    expect(activityForTab(state, 99, 1_500)).toMatchObject({
      createdAt: null,
      firstObservedAt: 1_500,
      activityDataCompleteness: "chrome_snapshot_only",
    });
  });
});

describe("activity reconciliation", () => {
  it("preserves current records across worker restart and deletes missing tabs", () => {
    let state = createActivityState("session-1", 1_000);
    state = reduceActivity(state, { type: "tab-created", tabId: 1, incognito: false }, 1_100);
    state = reduceActivity(state, { type: "tab-created", tabId: 2, incognito: false }, 1_200);
    state = reduceActivity(state, { type: "tab-activated", tabId: 1 }, 1_300);

    const restarted = reconcileActivity(
      state,
      [{ id: 1, incognito: false }],
      "session-1",
      1_400,
    );
    expect(restarted.records[1].activationCount).toBe(1);
    expect(restarted.records[2]).toBeUndefined();
  });

  it("does not carry metadata through a browser session change or tab ID reuse", () => {
    let state = createActivityState("old-session", 1_000);
    state = reduceActivity(state, { type: "tab-created", tabId: 1, incognito: false }, 1_100);
    state = reduceActivity(state, { type: "tab-activated", tabId: 1 }, 1_200);

    const next = reconcileActivity(
      state,
      [{ id: 1, incognito: false }],
      "new-session",
      2_000,
    );
    expect(next.sessionId).toBe("new-session");
    expect(next.records[1]).toMatchObject({
      createdAt: null,
      activationCount: 0,
      trackingSince: 2_000,
      activityDataCompleteness: "tracking_started_after_creation",
    });
  });
});
