import {
  createActivityState,
  reconcileActivity,
  reduceActivity,
  type ActivityEvent,
  type ActivityState,
  type CurrentTabIdentity,
} from "./activity-reducer";

const sessionKey = "browserSessionId";
const activityKey = "activityState";

export interface StorageAreaLike {
  get(key: string): Promise<Record<string, unknown>>;
  set(values: Record<string, unknown>): Promise<void>;
}

export async function getOrCreateBrowserSessionID(
  storage: StorageAreaLike,
  randomID: () => string = () => crypto.randomUUID(),
): Promise<string> {
  const stored = await storage.get(sessionKey);
  if (typeof stored[sessionKey] === "string" && stored[sessionKey] !== "") {
    return stored[sessionKey];
  }
  const sessionID = randomID();
  await storage.set({ [sessionKey]: sessionID });
  return sessionID;
}

export class ActivityRuntime {
  private state: ActivityState | undefined;
  private queued: Array<{ event: ActivityEvent; now: number }> = [];
  private flushScheduled = false;

  constructor(
    private readonly local: StorageAreaLike,
    private readonly session: StorageAreaLike,
    private readonly queryTabs: () => Promise<CurrentTabIdentity[]>,
  ) {}

  async initialize(now = Date.now()): Promise<void> {
    const [sessionID, stored, currentTabs] = await Promise.all([
      getOrCreateBrowserSessionID(this.session),
      this.local.get(activityKey),
      this.queryTabs(),
    ]);
    const previous = isActivityState(stored[activityKey])
      ? stored[activityKey]
      : createActivityState(sessionID, now);
    this.state = reconcileActivity(previous, currentTabs, sessionID, now);
    for (const queued of this.queued) {
      this.state = reduceActivity(this.state, queued.event, queued.now);
    }
    this.queued = [];
    await this.persist();
  }

  dispatch(event: ActivityEvent, now = Date.now()): void {
    if (this.state === undefined) {
      this.queued.push({ event, now });
      return;
    }
    this.state = reduceActivity(this.state, event, now);
    this.schedulePersist();
  }

  snapshot(): ActivityState | undefined {
    return this.state;
  }

  private schedulePersist(): void {
    if (this.flushScheduled) return;
    this.flushScheduled = true;
    queueMicrotask(() => {
      this.flushScheduled = false;
      void this.persist();
    });
  }

  private async persist(): Promise<void> {
    if (this.state !== undefined) {
      await this.local.set({ [activityKey]: this.state });
    }
  }
}

function isActivityState(value: unknown): value is ActivityState {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as ActivityState).sessionId === "string" &&
    typeof (value as ActivityState).trackingSince === "number" &&
    typeof (value as ActivityState).records === "object"
  );
}
