import { NativeConnectionManager } from "./native-connection";
import {
  registerActivityListeners,
  restrictStorageToTrustedContexts,
  type ActivityChromeEvents,
} from "./activity-events";
import { ActivityRuntime } from "./activity-store";
import {
  createHandshake,
  protocolVersion,
  validateEnvelope,
  validateHostHandshake,
  type NativeEnvelope,
} from "./protocol";
import { buildSnapshot } from "./snapshot";
import { ContentRevisionStore, invalidateContentRevision } from "./content";
import {
  compareTabContent,
  ContentServiceError,
  diffTabContent,
  getTabContent,
  validateCurrentContentRevisions,
} from "./content-service";
import {
  applyChromeOperation,
  restoreChromeSnapshot,
  type PlanOperation,
  type UndoSnapshot,
} from "./chrome-operations";
import { closeTabs, TabCloseError } from "./tab-close";

const contentRevisions = ContentRevisionStore.load(chrome.storage.session);

chrome.tabs.onUpdated.addListener((tabId, changeInfo) => {
  if (changeInfo.status === "loading" || changeInfo.url !== undefined) {
    void contentRevisions.then((store) =>
      invalidateContentRevision(store, {
        type: changeInfo.url === undefined ? "reload" : "url-changed",
        tabId,
      }),
    );
  }
});
chrome.tabs.onRemoved.addListener((tabId) =>
  void contentRevisions.then((store) =>
    invalidateContentRevision(store, { type: "tab-removed", tabId }),
  ),
);

const activityRuntime = new ActivityRuntime(
  chrome.storage.local,
  chrome.storage.session,
  async () =>
    (await chrome.tabs.query({})).flatMap((tab) =>
      tab.id === undefined ? [] : [{ id: tab.id, incognito: tab.incognito }],
    ),
);

// Register synchronously during module evaluation so MV3 event wakeups are not lost.
registerActivityListeners(
  { tabs: chrome.tabs, tabGroups: chrome.tabGroups } as unknown as ActivityChromeEvents,
  (event) => activityRuntime.dispatch(event),
);
void restrictStorageToTrustedContexts(chrome.storage.local);
void activityRuntime.initialize();

const nativeHostName = "io.github.masahide.tabcli";
const connection = new NativeConnectionManager(
  () => {
    const port = chrome.runtime.connectNative(nativeHostName);
	port.onDisconnect.addListener(() => {
		const message = chrome.runtime.lastError?.message;
		if (message) console.error("Native Messaging connection failed:", message);
	});
    port.onMessage.addListener((message: unknown) => {
      void handleNativeMessage(port, message);
    });
    port.postMessage(createHandshake());
    return port;
  },
  { initialDelayMs: 250, maxDelayMs: 4_000, maxRetries: 5 },
);

// Manifest V3 listeners must be registered synchronously at module evaluation.
chrome.runtime.onStartup.addListener(() => connection.start());
chrome.runtime.onInstalled.addListener(() => connection.start());

connection.start();

async function handleNativeMessage(
  port: chrome.runtime.Port,
  message: unknown,
): Promise<void> {
  try {
    validateEnvelope(message);
    if (message.operation === "handshake") {
      validateHostHandshake(message.payload);
      return;
    }
    if (message.operation === "content_get") {
      const settings = await chrome.storage.local.get("tabignore");
      const tabignore = Array.isArray(settings.tabignore)
        ? settings.tabignore.filter((value): value is string => typeof value === "string")
        : [];
      const response: NativeEnvelope = {
        protocolVersion,
        id: message.id,
        operation: message.operation,
        payload: await getTabContent(
          message.payload as { tabId: number; maxChars?: number },
          tabignore,
          await contentRevisions,
        ),
      };
      port.postMessage(response);
      return;
    }
    if (message.operation === "content_compare" || message.operation === "content_diff") {
      const settings = await chrome.storage.local.get("tabignore");
      const tabignore = Array.isArray(settings.tabignore)
        ? settings.tabignore.filter((value): value is string => typeof value === "string")
        : [];
      const payload =
        message.operation === "content_compare"
          ? await compareTabContent(message.payload as { tabIds: number[] }, tabignore)
          : await diffTabContent(
              message.payload as {
                tabIds: number[];
                maxChars?: number;
                maxDiffChars?: number;
              },
              tabignore,
            );
      port.postMessage({
        protocolVersion,
        id: message.id,
        operation: message.operation,
        payload,
      } satisfies NativeEnvelope);
      return;
    }
    if (message.operation === "content_revisions_validate") {
      const references = message.payload as Array<{ tabId: number; revision: string }>;
      const invalidTabIds = await validateCurrentContentRevisions(
        references,
        await contentRevisions,
      );
      port.postMessage({
        protocolVersion,
        id: message.id,
        operation: message.operation,
        payload: { valid: invalidTabIds.length === 0, invalidTabIds },
      } satisfies NativeEnvelope);
      return;
    }
    if (message.operation === "apply_operation") {
      port.postMessage({
        protocolVersion,
        id: message.id,
        operation: message.operation,
        payload: await applyChromeOperation(message.payload as PlanOperation),
      } satisfies NativeEnvelope);
      return;
    }
    if (message.operation === "tabs_close") {
      port.postMessage({
        protocolVersion,
        id: message.id,
        operation: message.operation,
        payload: await closeTabs(chrome.tabs, message.payload as { tabIds: number[] }),
      } satisfies NativeEnvelope);
      return;
    }
    if (message.operation === "rollback" || message.operation === "undo_restore") {
      const request = message.payload as { snapshot: UndoSnapshot };
      const restored = await restoreChromeSnapshot(request.snapshot);
      port.postMessage({
        protocolVersion,
        id: message.id,
        operation: message.operation,
        payload:
          message.operation === "rollback"
            ? { complete: restored.unrestorable.length === 0, unrestorable: restored.unrestorable }
            : restored,
      } satisfies NativeEnvelope);
      return;
    }
    if (message.operation !== "snapshot") return;
    const [tabs, groups, settings] = await Promise.all([
      chrome.tabs.query({ windowType: "normal" }),
      chrome.tabGroups.query({}),
      chrome.storage.local.get("tabignore"),
    ]);
    const tabignore = Array.isArray(settings.tabignore)
      ? settings.tabignore.filter((value): value is string => typeof value === "string")
      : [];
    const response: NativeEnvelope = {
      protocolVersion,
      id: message.id,
      operation: message.operation,
      payload: buildSnapshot(
        tabs,
        groups,
        tabignore,
        activityRuntime.snapshot(),
      ),
    };
    port.postMessage(response);
  } catch (error) {
    const request = message as Partial<NativeEnvelope>;
    const serviceError =
      error instanceof ContentServiceError || error instanceof TabCloseError
        ? error
        : undefined;
    port.postMessage({
      protocolVersion,
      id: typeof request.id === "string" ? request.id : "invalid-request",
      operation:
        typeof request.operation === "string" ? request.operation : "invalid",
      error: {
        code: serviceError?.code ?? "PROTOCOL_VERSION_MISMATCH",
        message: serviceError?.message ?? "Native Messaging request is incompatible",
        details: serviceError?.details,
      },
    } satisfies NativeEnvelope);
  }
}
