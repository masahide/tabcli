export type ActivityDataCompleteness =
  | "created_observed"
  | "tracking_started_after_creation"
  | "chrome_snapshot_only";

export interface ActivityRecord {
  createdAt: number | null;
  firstObservedAt: number;
  activationCount: number;
  lastMovedAt: number | null;
  lastGroupChangedAt: number | null;
  trackingSince: number;
  activityDataCompleteness: ActivityDataCompleteness;
}

export interface ActivityState {
  sessionId: string;
  trackingSince: number;
  records: Record<number, ActivityRecord>;
}

export type ActivityEvent =
  | { type: "tab-created"; tabId: number; incognito: boolean }
  | { type: "tab-activated"; tabId: number }
  | { type: "tab-updated"; tabId: number; groupChanged: boolean }
  | { type: "tab-moved" | "tab-attached" | "tab-detached"; tabId: number }
  | { type: "tab-removed"; tabId: number }
  | { type: "group-changed"; tabIds: number[] };

export interface CurrentTabIdentity {
  id: number;
  incognito: boolean;
}

export function createActivityState(
  sessionId: string,
  trackingSince: number,
): ActivityState {
  return { sessionId, trackingSince, records: {} };
}

function existingObservation(state: ActivityState, now: number): ActivityRecord {
  return {
    createdAt: null,
    firstObservedAt: now,
    activationCount: 0,
    lastMovedAt: null,
    lastGroupChangedAt: null,
    trackingSince: state.trackingSince,
    activityDataCompleteness: "tracking_started_after_creation",
  };
}

function updateRecord(
  state: ActivityState,
  tabId: number,
  now: number,
  update: (record: ActivityRecord) => ActivityRecord,
): ActivityState {
  const record = state.records[tabId] ?? existingObservation(state, now);
  return {
    ...state,
    records: { ...state.records, [tabId]: update(record) },
  };
}

export function reduceActivity(
  state: ActivityState,
  event: ActivityEvent,
  now: number,
): ActivityState {
  switch (event.type) {
    case "tab-created":
      if (event.incognito) return state;
      return {
        ...state,
        records: {
          ...state.records,
          [event.tabId]: {
            createdAt: now,
            firstObservedAt: now,
            activationCount: 0,
            lastMovedAt: null,
            lastGroupChangedAt: null,
            trackingSince: state.trackingSince,
            activityDataCompleteness: "created_observed",
          },
        },
      };
    case "tab-activated":
      return updateRecord(state, event.tabId, now, (record) => ({
        ...record,
        activationCount: record.activationCount + 1,
      }));
    case "tab-updated":
      if (!event.groupChanged) return state;
      return updateRecord(state, event.tabId, now, (record) => ({
        ...record,
        lastGroupChangedAt: now,
      }));
    case "tab-moved":
    case "tab-attached":
    case "tab-detached":
      return updateRecord(state, event.tabId, now, (record) => ({
        ...record,
        lastMovedAt: now,
      }));
    case "tab-removed": {
      if (state.records[event.tabId] === undefined) return state;
      const records = { ...state.records };
      delete records[event.tabId];
      return { ...state, records };
    }
    case "group-changed":
      return event.tabIds.reduce(
        (current, tabId) =>
          updateRecord(current, tabId, now, (record) => ({
            ...record,
            lastGroupChangedAt: now,
          })),
        state,
      );
  }
}

export function reconcileActivity(
  previous: ActivityState,
  currentTabs: CurrentTabIdentity[],
  sessionId: string,
  now: number,
): ActivityState {
  const state =
    previous.sessionId === sessionId
      ? { ...previous, records: { ...previous.records } }
      : createActivityState(sessionId, now);
  const currentIDs = new Set(
    currentTabs.filter((tab) => !tab.incognito).map((tab) => tab.id),
  );
  for (const key of Object.keys(state.records)) {
    const tabId = Number(key);
    if (!currentIDs.has(tabId)) delete state.records[tabId];
  }
  for (const tabId of currentIDs) {
    if (state.records[tabId] === undefined) {
      state.records[tabId] = existingObservation(state, now);
    }
  }
  return state;
}

export function activityForTab(
  state: ActivityState,
  tabId: number,
  now: number,
): ActivityRecord {
  return (
    state.records[tabId] ?? {
      createdAt: null,
      firstObservedAt: now,
      activationCount: 0,
      lastMovedAt: null,
      lastGroupChangedAt: null,
      trackingSince: state.trackingSince,
      activityDataCompleteness: "chrome_snapshot_only",
    }
  );
}
