import type { ActivityEvent } from "./activity-reducer";

interface EventLike {
  addListener(listener: (...args: any[]) => void): void;
}

export interface ActivityChromeEvents {
  tabs: {
    onCreated: EventLike;
    onActivated: EventLike;
    onUpdated: EventLike;
    onMoved: EventLike;
    onAttached: EventLike;
    onDetached: EventLike;
    onRemoved: EventLike;
  };
  tabGroups: {
    onCreated: EventLike;
    onUpdated: EventLike;
    onMoved: EventLike;
    onRemoved: EventLike;
  };
}

export function registerActivityListeners(
  api: ActivityChromeEvents,
  dispatch: (event: ActivityEvent) => void,
): void {
  api.tabs.onCreated.addListener((tab: { id?: number; incognito: boolean }) => {
    if (tab.id !== undefined) dispatch({ type: "tab-created", tabId: tab.id, incognito: tab.incognito });
  });
  api.tabs.onActivated.addListener(({ tabId }: { tabId: number }) => dispatch({ type: "tab-activated", tabId }));
  api.tabs.onUpdated.addListener((tabId: number, change: { groupId?: number }) =>
    dispatch({ type: "tab-updated", tabId, groupChanged: change.groupId !== undefined }),
  );
  api.tabs.onMoved.addListener((tabId: number) => dispatch({ type: "tab-moved", tabId }));
  api.tabs.onAttached.addListener((tabId: number) => dispatch({ type: "tab-attached", tabId }));
  api.tabs.onDetached.addListener((tabId: number) => dispatch({ type: "tab-detached", tabId }));
  api.tabs.onRemoved.addListener((tabId: number) => dispatch({ type: "tab-removed", tabId }));
  api.tabGroups.onCreated.addListener(() => dispatch({ type: "group-changed", tabIds: [] }));
  api.tabGroups.onUpdated.addListener(() => dispatch({ type: "group-changed", tabIds: [] }));
  api.tabGroups.onMoved.addListener(() => dispatch({ type: "group-changed", tabIds: [] }));
  api.tabGroups.onRemoved.addListener(() => dispatch({ type: "group-changed", tabIds: [] }));
}

export function restrictStorageToTrustedContexts(storage: {
  setAccessLevel(options: { accessLevel: "TRUSTED_CONTEXTS" }): Promise<void>;
}): Promise<void> {
  return storage.setAccessLevel({ accessLevel: "TRUSTED_CONTEXTS" });
}
