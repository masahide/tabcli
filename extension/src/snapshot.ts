export interface ChromeTabLike {
  id?: number;
  windowId: number;
  index: number;
  title?: string;
  url?: string;
  active: boolean;
  pinned: boolean;
  groupId: number;
  incognito: boolean;
  lastAccessed?: number;
}

export interface ChromeGroupLike {
  id: number;
  windowId: number;
  title?: string;
  color: string;
  collapsed: boolean;
}

export interface SnapshotTab {
  id: number;
  windowId: number;
  index: number;
  title: string;
  url: string;
  active: boolean;
  pinned: boolean;
  groupId: number;
  lastAccessed: number | null;
  operable: boolean;
  activity: ActivityRecord;
}

export interface SnapshotGroup extends ChromeGroupLike {
  title: string;
  tabIds: number[];
}

export interface Snapshot {
  sessionId: string;
  tabs: SnapshotTab[];
  groups: SnapshotGroup[];
}

const explicitPattern = /^(\*|https?|file|ftp):\/\/(\*|\*\.[A-Za-z0-9.-]+|[A-Za-z0-9.-]+)(\/.*)$/;
const shorthandHost = /^(\*\.)?[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+$/;

export function normalizeTabIgnorePattern(input: string): string {
  const value = input.trim();
  if (explicitPattern.test(value)) return value;
  if (shorthandHost.test(value)) return `*://${value}/*`;
  throw new Error("tabignore must be a Chrome match pattern or a hostname");
}

export function patternMatches(pattern: string, urlValue: string): boolean {
  const match = explicitPattern.exec(normalizeTabIgnorePattern(pattern));
  if (!match) return false;
  let url: URL;
  try {
    url = new URL(urlValue);
  } catch {
    return false;
  }
  const [, scheme, host, path] = match;
  if (scheme !== "*" && `${scheme}:` !== url.protocol) return false;
  if (scheme === "*" && url.protocol !== "http:" && url.protocol !== "https:") return false;
  if (host !== "*") {
    if (host.startsWith("*.")) {
      const suffix = host.slice(2);
      if (url.hostname !== suffix && !url.hostname.endsWith(`.${suffix}`)) return false;
    } else if (url.hostname !== host) {
      return false;
    }
  }
  const pathExpression = path
    .replace(/[.+?^${}()|[\]\\]/g, "\\$&")
    .replaceAll("*", ".*");
  return new RegExp(`^${pathExpression}$`).test(url.pathname + url.search);
}

function isOperable(url: string): boolean {
  return url.startsWith("http://") || url.startsWith("https://") || url.startsWith("file://");
}

export function buildSnapshot(
  tabs: ChromeTabLike[],
  groups: ChromeGroupLike[],
  tabIgnore: string[],
  activityState?: ActivityState,
  now = Date.now(),
): Snapshot {
  const patterns = tabIgnore.map(normalizeTabIgnorePattern);
  const visibleTabs = tabs
    .filter((tab): tab is ChromeTabLike & { id: number } => tab.id !== undefined && !tab.incognito)
    .filter((tab) => !patterns.some((pattern) => patternMatches(pattern, tab.url ?? "")))
    .map((tab): SnapshotTab => ({
      id: tab.id,
      windowId: tab.windowId,
      index: tab.index,
      title: tab.title ?? "",
      url: tab.url ?? "",
      active: tab.active,
      pinned: tab.pinned,
      groupId: tab.groupId,
      lastAccessed: tab.lastAccessed === undefined ? null : Math.trunc(tab.lastAccessed),
      operable: isOperable(tab.url ?? ""),
      activity: activityState
        ? activityForTab(activityState, tab.id, now)
        : {
            createdAt: null,
            firstObservedAt: now,
            activationCount: 0,
            lastMovedAt: null,
            lastGroupChangedAt: null,
            trackingSince: now,
            activityDataCompleteness: "chrome_snapshot_only",
          },
    }));

  const tabIDsByGroup = new Map<number, number[]>();
  for (const tab of visibleTabs) {
    if (tab.groupId < 0) continue;
    const ids = tabIDsByGroup.get(tab.groupId) ?? [];
    ids.push(tab.id);
    tabIDsByGroup.set(tab.groupId, ids);
  }
  const visibleGroups = groups
    .filter((group) => tabIDsByGroup.has(group.id))
    .map((group): SnapshotGroup => ({
      ...group,
      title: group.title ?? "",
      tabIds: tabIDsByGroup.get(group.id) ?? [],
    }));

  return {
    sessionId: activityState?.sessionId ?? "unknown",
    tabs: visibleTabs,
    groups: visibleGroups,
  };
}
import {
  activityForTab,
  type ActivityRecord,
  type ActivityState,
} from "./activity-reducer";
