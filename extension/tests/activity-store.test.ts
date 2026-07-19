import { describe, expect, it, vi } from "vitest";

import { ActivityRuntime, getOrCreateBrowserSessionID } from "../src/activity-store";

describe("browser session ID", () => {
  it("creates and stores one ID for a new browser session", async () => {
    const storage = {
      get: vi.fn().mockResolvedValue({}),
      set: vi.fn().mockResolvedValue(undefined),
    };
    await expect(
      getOrCreateBrowserSessionID(storage, () => "session-new"),
    ).resolves.toBe("session-new");
    expect(storage.set).toHaveBeenCalledWith({
      browserSessionId: "session-new",
    });
  });

  it("reuses the session storage ID after a worker restart", async () => {
    const storage = {
      get: vi.fn().mockResolvedValue({ browserSessionId: "session-existing" }),
      set: vi.fn().mockResolvedValue(undefined),
    };
    await expect(getOrCreateBrowserSessionID(storage)).resolves.toBe(
      "session-existing",
    );
    expect(storage.set).not.toHaveBeenCalled();
  });
});

it("reconciles saved metadata with current tabs during extension startup", async () => {
  const local = {
    get: vi.fn().mockResolvedValue({
      activityState: {
        sessionId: "session-1",
        trackingSince: 1_000,
        records: {
          1: {
            createdAt: 1_100,
            firstObservedAt: 1_100,
            activationCount: 2,
            lastMovedAt: null,
            lastGroupChangedAt: null,
            trackingSince: 1_000,
            activityDataCompleteness: "created_observed",
          },
          2: {
            createdAt: 1_200,
            firstObservedAt: 1_200,
            activationCount: 0,
            lastMovedAt: null,
            lastGroupChangedAt: null,
            trackingSince: 1_000,
            activityDataCompleteness: "created_observed",
          },
        },
      },
    }),
    set: vi.fn().mockResolvedValue(undefined),
  };
  const session = {
    get: vi.fn().mockResolvedValue({ browserSessionId: "session-1" }),
    set: vi.fn().mockResolvedValue(undefined),
  };
  const runtime = new ActivityRuntime(local, session, async () => [
    { id: 1, incognito: false },
  ]);

  await runtime.initialize(2_000);

  expect(runtime.snapshot()?.records[1].activationCount).toBe(2);
  expect(runtime.snapshot()?.records[2]).toBeUndefined();
  expect(local.set).toHaveBeenCalledTimes(1);
});

it("coalesces activity writes from the same tick", async () => {
  const local = {
    get: vi.fn().mockResolvedValue({}),
    set: vi.fn().mockResolvedValue(undefined),
  };
  const session = {
    get: vi.fn().mockResolvedValue({ browserSessionId: "session-1" }),
    set: vi.fn().mockResolvedValue(undefined),
  };
  const runtime = new ActivityRuntime(local, session, async () => []);
  await runtime.initialize(1_000);
  expect(local.set).toHaveBeenCalledTimes(1);

  runtime.dispatch({ type: "tab-created", tabId: 1, incognito: false }, 1_100);
  runtime.dispatch({ type: "tab-activated", tabId: 1 }, 1_200);
  runtime.dispatch({ type: "tab-moved", tabId: 1 }, 1_300);
  await Promise.resolve();

  expect(local.set).toHaveBeenCalledTimes(2);
});
