export interface PlanOperation {
  kind: "create_group" | "update_group" | "move_tab" | "ungroup_tab";
  tabId?: number;
  groupId?: number;
  newGroupKey?: string;
  windowId?: number;
  title?: string;
  color?: string;
  collapsed?: boolean;
  pinned?: boolean;
  index?: number;
}

export interface AppliedOperation {
  operation: PlanOperation;
  createdGroupId?: number;
}

export interface UndoSnapshot {
  sessionId: string;
  tabs: Array<{
    tabId: number;
    windowId: number;
    index: number;
    pinned: boolean;
    groupId: number;
  }>;
  groups: Array<{
    groupId: number;
    windowId: number;
    title: string;
    color: string;
    collapsed: boolean;
  }>;
}

export interface RestoreResult {
  status: "success" | "partial";
  restoredTabIds: number[];
  restoredGroupIds: number[];
  unrestorable: string[];
}

export async function applyChromeOperation(
  operation: PlanOperation,
): Promise<AppliedOperation> {
  switch (operation.kind) {
    case "create_group": {
      if (operation.tabId === undefined || operation.windowId === undefined) {
        throw new Error("create_group requires tabId and windowId");
      }
      const groupId = await chrome.tabs.group({
        tabIds: [operation.tabId],
        createProperties: { windowId: operation.windowId },
      });
      await chrome.tabGroups.update(groupId, {
        title: operation.title,
        color: (operation.color ?? "grey") as NonNullable<chrome.tabGroups.UpdateProperties["color"]>,
      });
      return { operation, createdGroupId: groupId };
    }
    case "update_group":
      if (operation.groupId === undefined) throw new Error("update_group requires groupId");
      await chrome.tabGroups.update(operation.groupId, {
        ...(operation.title === undefined || operation.title === "" ? {} : { title: operation.title }),
        ...(operation.color === undefined || operation.color === "" ? {} : { color: operation.color as NonNullable<chrome.tabGroups.UpdateProperties["color"]> }),
        ...(operation.collapsed === undefined ? {} : { collapsed: operation.collapsed }),
      });
      return { operation };
    case "move_tab":
      if (operation.tabId === undefined) throw new Error("move_tab requires tabId");
      if (operation.groupId !== undefined && operation.groupId >= 0) {
        await chrome.tabs.group({ tabIds: [operation.tabId], groupId: operation.groupId });
      }
      await applyTabPosition(operation);
      return { operation };
    case "ungroup_tab":
      if (operation.tabId === undefined) throw new Error("ungroup_tab requires tabId");
      await chrome.tabs.ungroup([operation.tabId]);
      await applyTabPosition(operation);
      return { operation };
  }
}

async function applyTabPosition(operation: PlanOperation): Promise<void> {
  if (operation.tabId === undefined) return;
  if (operation.pinned !== undefined) {
    await chrome.tabs.update(operation.tabId, { pinned: operation.pinned });
  }
  if (operation.index !== undefined) {
    await chrome.tabs.move(operation.tabId, { index: operation.index });
  }
}

export async function restoreChromeSnapshot(snapshot: UndoSnapshot): Promise<RestoreResult> {
  const currentTabs = await chrome.tabs.query({});
  const currentTabIDs = new Set(currentTabs.flatMap((tab) => (tab.id === undefined ? [] : [tab.id])));
  const currentGroups = await chrome.tabGroups.query({});
  const currentGroupIDs = new Set(currentGroups.map((group) => group.id));
  const groupMapping = new Map<number, number>();
  const restoredGroupIds: number[] = [];
  const unrestorable: string[] = [];

  for (const group of snapshot.groups) {
    const memberIDs = snapshot.tabs
      .filter((tab) => tab.groupId === group.groupId && currentTabIDs.has(tab.tabId))
      .map((tab) => tab.tabId);
    let groupID = group.groupId;
    if (!currentGroupIDs.has(groupID)) {
      if (memberIDs.length === 0) {
        unrestorable.push(`group:${group.groupId}:deleted`);
        continue;
      }
      groupID = await chrome.tabs.group({
        tabIds: memberIDs as [number, ...number[]],
        createProperties: { windowId: group.windowId },
      });
    }
    groupMapping.set(group.groupId, groupID);
    await chrome.tabGroups.update(groupID, {
      title: group.title,
      color: group.color as NonNullable<chrome.tabGroups.UpdateProperties["color"]>,
      collapsed: group.collapsed,
    });
    restoredGroupIds.push(groupID);
  }

  const restoredTabIds: number[] = [];
  for (const tab of snapshot.tabs) {
    if (!currentTabIDs.has(tab.tabId)) {
      unrestorable.push(`tab:${tab.tabId}:closed`);
      continue;
    }
    if (tab.groupId < 0) {
      const current = currentTabs.find((candidate) => candidate.id === tab.tabId);
      if (current !== undefined && current.groupId >= 0) await chrome.tabs.ungroup([tab.tabId]);
    } else {
      const groupID = groupMapping.get(tab.groupId);
      if (groupID === undefined) {
        unrestorable.push(`tab:${tab.tabId}:group-unavailable`);
        continue;
      }
      await chrome.tabs.group({ tabIds: [tab.tabId], groupId: groupID });
    }
    await chrome.tabs.update(tab.tabId, { pinned: tab.pinned });
    restoredTabIds.push(tab.tabId);
  }
  for (const tab of [...snapshot.tabs].sort((left, right) => left.windowId - right.windowId || left.index - right.index)) {
    if (currentTabIDs.has(tab.tabId)) {
      await chrome.tabs.move(tab.tabId, { windowId: tab.windowId, index: tab.index });
    }
  }
  return {
    status: unrestorable.length === 0 ? "success" : "partial",
    restoredTabIds,
    restoredGroupIds,
    unrestorable,
  };
}
