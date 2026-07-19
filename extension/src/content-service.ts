import {
  ContentRevisionStore,
  contentAccessError,
  createLineDiff,
  defaultDiffOutputLimit,
  defaultDiffSourceLimit,
  fixedIdentityProbe,
  fixedPageDiffSnapshot,
  fixedPageExtractor,
  fixedPageHasher,
  mapExtractionError,
  validateStableIdentity,
  type LineChange,
  type PageDiffSnapshot,
  type PageHash,
} from "./content";
import { patternMatches } from "./snapshot";
import { originPermissionPattern } from "./settings";

export interface ContentGetRequest {
  tabId: number;
  maxChars?: number;
}

export interface ContentGetResponse {
  tabId: number;
  title: string;
  url: string;
  contentType: "text/plain";
  text: string;
  extractedAt: string;
  truncated: boolean;
  originalCharCount: number;
  returnedCharCount: number;
  untrustedContent: true;
  contentRevision: string;
  dataHandlingNotice: string;
}

export interface ContentCompareRequest {
  tabIds: number[];
}

export interface ContentHashTab {
  tabId: number;
  title: string;
  url: string;
  sha256: string;
  characterCount: number;
}

export interface ContentCompareResponse {
  hashAlgorithm: "SHA-256";
  match: boolean;
  comparedAt: string;
  tabs: [ContentHashTab, ContentHashTab];
  dataHandlingNotice: string;
}

export interface ContentDiffRequest {
  tabIds: number[];
  maxChars?: number;
  maxDiffChars?: number;
}

export interface ContentDiffTab extends ContentHashTab {
  sourceTruncated: boolean;
  returnedCharacterCount: number;
}

export interface ContentDiffResponse {
  hashAlgorithm: "SHA-256";
  diffAlgorithm: "line-lcs-or-bounded-replacement";
  format: "line-changes";
  match: boolean;
  comparedAt: string;
  tabs: [ContentDiffTab, ContentDiffTab];
  changes: LineChange[];
  untrustedContent: true;
  minimal: boolean;
  sourceTruncated: boolean;
  diffTruncated: boolean;
  originalChangeCount: number;
  returnedChangeCount: number;
  originalDiffCharacterCount: number;
  returnedDiffCharacterCount: number;
  dataHandlingNotice: string;
}

export class ContentServiceError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly details?: Record<string, unknown>,
  ) {
    super(message);
  }
}

export async function validateCurrentContentRevisions(
  references: Array<{ tabId: number; revision: string }>,
  revisions: ContentRevisionStore,
): Promise<number[]> {
  const invalid: number[] = [];
  for (const reference of references) {
    const stored = revisions.identity(reference.revision);
    if (stored === undefined || stored.tabId !== reference.tabId) {
      invalid.push(reference.tabId);
      continue;
    }
    try {
      const results = await chrome.scripting.executeScript({
        target: { tabId: reference.tabId, allFrames: false },
        world: "ISOLATED",
        func: fixedIdentityProbe,
      });
      if (
        results.length !== 1 ||
        results[0].result === undefined ||
        results[0].documentId === undefined ||
        !revisions.isValid(reference.revision, {
          tabId: reference.tabId,
          url: results[0].result.url,
          documentId: results[0].documentId,
        })
      ) {
        invalid.push(reference.tabId);
      }
    } catch {
      invalid.push(reference.tabId);
    }
  }
  return invalid;
}

export async function getTabContent(
  request: ContentGetRequest,
  tabignore: string[],
  revisions: ContentRevisionStore,
): Promise<ContentGetResponse> {
  const tab = await requireAccessibleTab(request.tabId, tabignore);
  const url = tab.url ?? "";
  const maxCharacters = request.maxChars ?? 10_000;

  let extractedResult: ReturnType<typeof fixedPageExtractor>;
  let extractionDocumentID: string | undefined;
  try {
    const results = await chrome.scripting.executeScript({
      target: { tabId: request.tabId, allFrames: false },
      world: "ISOLATED",
      func: fixedPageExtractor,
      args: [maxCharacters],
    });
    if (results.length !== 1 || results[0].result === undefined) {
      throw new Error("main frame did not return a result");
    }
    extractedResult = results[0].result;
    extractionDocumentID = results[0].documentId;
  } catch (error) {
    const mapped = mapExtractionError(error);
    throw new ContentServiceError(mapped.code, mapped.message);
  }

  if (extractionDocumentID === undefined) {
    throw new ContentServiceError("CONTENT_STALE", "The page changed during extraction");
  }
  const identity = {
    tabId: request.tabId,
    url: extractedResult.url,
    documentId: extractionDocumentID,
  };
  const latestTab = await validateCapturedPage({
    tabId: request.tabId,
    result: { url: extractedResult.url },
    documentId: extractionDocumentID,
  });
  return {
    tabId: request.tabId,
    title: latestTab.title ?? "",
    url: identity.url,
    contentType: "text/plain",
    text: extractedResult.extracted.text,
    extractedAt: new Date().toISOString(),
    truncated: extractedResult.extracted.truncated,
    originalCharCount: extractedResult.extracted.originalCharacterCount,
    returnedCharCount: extractedResult.extracted.returnedCharacterCount,
    untrustedContent: true,
    contentRevision: revisions.issue(identity),
    dataHandlingNotice:
      "Page text is returned through the MCP client and may be sent to the model provider configured by the user. The extension and tabcli do not persist it.",
  };
}

interface CapturedPage<T extends { url: string }> {
  tabId: number;
  result: T;
  documentId: string;
}

export async function compareTabContent(
  request: ContentCompareRequest,
  tabignore: string[],
): Promise<ContentCompareResponse> {
  const tabIds = validateComparisonTabIDs(request.tabIds);
  await Promise.all(tabIds.map((tabId) => requireAccessibleTab(tabId, tabignore)));
  const captured = await Promise.all(tabIds.map((tabId) => capturePageHash(tabId)));
  const latestTabs = await Promise.all(captured.map(validateCapturedPage));
  const tabs = captured.map((page, index): ContentHashTab => ({
    tabId: page.tabId,
    title: latestTabs[index].title ?? "",
    url: page.result.url,
    sha256: page.result.sha256,
    characterCount: page.result.characterCount,
  })) as [ContentHashTab, ContentHashTab];
  return {
    hashAlgorithm: "SHA-256",
    match: tabs[0].sha256 === tabs[1].sha256,
    comparedAt: new Date().toISOString(),
    tabs,
    dataHandlingNotice:
      "Visible text was SHA-256 hashed inside the extension. Page text was not returned through Native Messaging and was not persisted.",
  };
}

export async function diffTabContent(
  request: ContentDiffRequest,
  tabignore: string[],
): Promise<ContentDiffResponse> {
  const tabIds = validateComparisonTabIDs(request.tabIds);
  const maxCharacters = request.maxChars ?? defaultDiffSourceLimit;
  const maxDiffCharacters = request.maxDiffChars ?? defaultDiffOutputLimit;
  if (!Number.isInteger(maxCharacters) || maxCharacters < 1 || maxCharacters > 50_000) {
    throw new ContentServiceError("INVALID_ARGUMENT", "maxChars must be between 1 and 50000");
  }
  if (!Number.isInteger(maxDiffCharacters) || maxDiffCharacters < 1 || maxDiffCharacters > 50_000) {
    throw new ContentServiceError("INVALID_ARGUMENT", "maxDiffChars must be between 1 and 50000");
  }

  await Promise.all(tabIds.map((tabId) => requireAccessibleTab(tabId, tabignore)));
  const captured = await Promise.all(
    tabIds.map((tabId) => capturePageDiffSnapshot(tabId, maxCharacters)),
  );
  const latestTabs = await Promise.all(captured.map(validateCapturedPage));
  const lineDiff = createLineDiff(
    captured[0].result.text,
    captured[1].result.text,
    maxDiffCharacters,
  );
  const tabs = captured.map((page, index): ContentDiffTab => ({
    tabId: page.tabId,
    title: latestTabs[index].title ?? "",
    url: page.result.url,
    sha256: page.result.sha256,
    characterCount: page.result.characterCount,
    sourceTruncated: page.result.truncated,
    returnedCharacterCount: page.result.returnedCharacterCount,
  })) as [ContentDiffTab, ContentDiffTab];
  return {
    hashAlgorithm: "SHA-256",
    diffAlgorithm: "line-lcs-or-bounded-replacement",
    format: "line-changes",
    match: tabs[0].sha256 === tabs[1].sha256,
    comparedAt: new Date().toISOString(),
    tabs,
    changes: lineDiff.changes,
    untrustedContent: true,
    minimal: lineDiff.minimal,
    sourceTruncated: tabs.some((tab) => tab.sourceTruncated),
    diffTruncated: lineDiff.truncated,
    originalChangeCount: lineDiff.originalChangeCount,
    returnedChangeCount: lineDiff.returnedChangeCount,
    originalDiffCharacterCount: lineDiff.originalCharacterCount,
    returnedDiffCharacterCount: lineDiff.returnedCharacterCount,
    dataHandlingNotice:
      "Only changed visible-text lines are returned through Native Messaging. Unchanged text and source snapshots are not returned or persisted.",
  };
}

function validateComparisonTabIDs(tabIds: number[]): [number, number] {
  if (
    !Array.isArray(tabIds) ||
    tabIds.length !== 2 ||
    !tabIds.every((tabId) => Number.isInteger(tabId) && tabId > 0) ||
    tabIds[0] === tabIds[1]
  ) {
    throw new ContentServiceError(
      "INVALID_ARGUMENT",
      "tabIds must contain exactly two distinct positive tab IDs",
    );
  }
  return [tabIds[0], tabIds[1]];
}

async function requireAccessibleTab(
  tabId: number,
  tabignore: string[],
): Promise<chrome.tabs.Tab> {
  let tab: chrome.tabs.Tab;
  try {
    tab = await chrome.tabs.get(tabId);
  } catch {
    throw new ContentServiceError("TAB_NOT_FOUND", "The selected tab is no longer available", {
      tabId,
    });
  }
  const url = tab.url ?? "";
  const ignored = tabignore.some((pattern) => patternMatches(pattern, url));
  const inaccessible = contentAccessError({
    url,
    incognito: tab.incognito,
    ignored,
    hasPermission: true,
  });
  if (inaccessible !== undefined) {
    throw new ContentServiceError(inaccessible, "Page text is not accessible for this tab");
  }
  const originPattern = originPermissionPattern(url);
  if (!(await chrome.permissions.contains({ origins: [originPattern] }))) {
    throw new ContentServiceError(
      "CONTENT_PERMISSION_REQUIRED",
      "Chrome site access is disabled for this origin",
      {
        origin: new URL(url).origin,
        action:
          "Open the extension details in chrome://extensions and allow site access for this origin or all sites.",
      },
    );
  }
  return tab;
}

async function capturePageHash(tabId: number): Promise<CapturedPage<PageHash>> {
  try {
    const results = await chrome.scripting.executeScript({
      target: { tabId, allFrames: false },
      world: "ISOLATED",
      func: fixedPageHasher,
    });
    if (
      results.length !== 1 ||
      results[0].result === undefined ||
      results[0].documentId === undefined
    ) {
      throw new Error("main frame did not return a result");
    }
    return { tabId, result: results[0].result, documentId: results[0].documentId };
  } catch (error) {
    const mapped = mapExtractionError(error);
    throw new ContentServiceError(mapped.code, mapped.message);
  }
}

async function capturePageDiffSnapshot(
  tabId: number,
  maxCharacters: number,
): Promise<CapturedPage<PageDiffSnapshot>> {
  try {
    const results = await chrome.scripting.executeScript({
      target: { tabId, allFrames: false },
      world: "ISOLATED",
      func: fixedPageDiffSnapshot,
      args: [maxCharacters],
    });
    if (
      results.length !== 1 ||
      results[0].result === undefined ||
      results[0].documentId === undefined
    ) {
      throw new Error("main frame did not return a result");
    }
    return { tabId, result: results[0].result, documentId: results[0].documentId };
  } catch (error) {
    const mapped = mapExtractionError(error);
    throw new ContentServiceError(mapped.code, mapped.message);
  }
}

async function validateCapturedPage<T extends { url: string }>(
  captured: CapturedPage<T>,
): Promise<chrome.tabs.Tab> {
  try {
    const results = await chrome.scripting.executeScript({
      target: { tabId: captured.tabId, allFrames: false },
      world: "ISOLATED",
      func: fixedIdentityProbe,
    });
    if (
      results.length !== 1 ||
      results[0].result === undefined ||
      results[0].documentId === undefined
    ) {
      throw new Error("identity probe did not return a result");
    }
    validateStableIdentity(
      { tabId: captured.tabId, url: captured.result.url, documentId: captured.documentId },
      { tabId: captured.tabId, url: results[0].result.url, documentId: results[0].documentId },
    );
    const latestTab = await chrome.tabs.get(captured.tabId);
    if (latestTab.url !== captured.result.url) {
      throw new Error("CONTENT_STALE");
    }
    return latestTab;
  } catch {
    throw new ContentServiceError("CONTENT_STALE", "The page changed during content processing");
  }
}
