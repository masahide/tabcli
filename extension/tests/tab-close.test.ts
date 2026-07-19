import { describe, expect, it, vi } from "vitest";

import { closeTabs, TabCloseError, type TabsRemoveAPI } from "../src/tab-close";
import { protocolVersion } from "../src/protocol";

describe("closeTabs", () => {
  it("closes exactly the selected tabs", async () => {
    const tabs: TabsRemoveAPI = { remove: vi.fn().mockResolvedValue(undefined) };

    await expect(closeTabs(tabs, { tabIds: [7, 9] })).resolves.toEqual({
      protocolVersion,
      closedTabIds: [7, 9],
    });
    expect(tabs.remove).toHaveBeenCalledWith([7, 9]);
  });

  it.each([
    { tabIds: [] },
    { tabIds: [0] },
    { tabIds: [7, 7] },
    { tabIds: [1.5] },
  ])("rejects invalid tab IDs $tabIds", async ({ tabIds }) => {
    const tabs: TabsRemoveAPI = { remove: vi.fn() };

    await expect(closeTabs(tabs, { tabIds })).rejects.toMatchObject({
      code: "INVALID_ARGUMENT",
    } satisfies Partial<TabCloseError>);
    expect(tabs.remove).not.toHaveBeenCalled();
  });

  it("returns a stable error when Chrome rejects the close", async () => {
    const tabs: TabsRemoveAPI = {
      remove: vi.fn().mockRejectedValue(new Error("No tab with id")),
    };

    await expect(closeTabs(tabs, { tabIds: [7] })).rejects.toMatchObject({
      code: "TAB_CLOSE_FAILED",
      details: { tabIds: [7] },
    } satisfies Partial<TabCloseError>);
  });
});
