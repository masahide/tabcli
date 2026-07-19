import { describe, expect, it, vi } from "vitest";

import {
  registerActivityListeners,
  restrictStorageToTrustedContexts,
  type ActivityChromeEvents,
} from "../src/activity-events";

function event() {
  return { addListener: vi.fn() };
}

describe("activity event registration", () => {
  it("registers every required listener synchronously", () => {
    const api: ActivityChromeEvents = {
      tabs: {
        onCreated: event(),
        onActivated: event(),
        onUpdated: event(),
        onMoved: event(),
        onAttached: event(),
        onDetached: event(),
        onRemoved: event(),
      },
      tabGroups: {
        onCreated: event(),
        onUpdated: event(),
        onMoved: event(),
        onRemoved: event(),
      },
    };

    registerActivityListeners(api, vi.fn());

    for (const listener of [
      ...Object.values(api.tabs),
      ...Object.values(api.tabGroups),
    ]) {
      expect(listener.addListener).toHaveBeenCalledTimes(1);
    }
  });

  it("restricts local storage to trusted extension contexts", async () => {
    const setAccessLevel = vi.fn().mockResolvedValue(undefined);
    await restrictStorageToTrustedContexts({ setAccessLevel });
    expect(setAccessLevel).toHaveBeenCalledWith({
      accessLevel: "TRUSTED_CONTEXTS",
    });
  });
});
