export const defaultContentLimit = 10_000;
export const maximumContentLimit = 50_000;
export const defaultDiffSourceLimit = 50_000;
export const defaultDiffOutputLimit = 20_000;
export const maximumDiffOutputLimit = 50_000;
export const maximumDiffEntries = 2_000;
const maximumLCSCells = 2_000_000;

export interface DocumentLike {
  body: { innerText: string } | null;
}

export interface ExtractedText {
  text: string;
  truncated: boolean;
  originalCharacterCount: number;
  returnedCharacterCount: number;
}

export interface ContentIdentity {
  tabId: number;
  url: string;
  documentId: string;
}

export interface PageHash {
  url: string;
  sha256: string;
  characterCount: number;
}

export interface PageDiffSnapshot extends PageHash {
  text: string;
  truncated: boolean;
  returnedCharacterCount: number;
}

export interface LineChange {
  kind: "delete" | "insert";
  oldLine?: number;
  newLine?: number;
  text: string;
  textTruncated?: true;
}

export interface LineDiff {
  changes: LineChange[];
  minimal: boolean;
  truncated: boolean;
  originalChangeCount: number;
  returnedChangeCount: number;
  originalCharacterCount: number;
  returnedCharacterCount: number;
}

export type ContentInvalidationEvent = {
  type: "navigation" | "reload" | "url-changed" | "tab-removed";
  tabId: number;
};

export function extractMainFrameVisibleText(
  documentLike: DocumentLike,
  maxCharacters = defaultContentLimit,
): ExtractedText {
  if (!Number.isInteger(maxCharacters) || maxCharacters < 1 || maxCharacters > maximumContentLimit) {
    throw new Error("maxCharacters must be between 1 and 50000");
  }
  const original = documentLike.body?.innerText ?? "";
  const text = original.slice(0, maxCharacters);
  return {
    text,
    truncated: original.length > maxCharacters,
    originalCharacterCount: original.length,
    returnedCharacterCount: text.length,
  };
}

export function contentAccessError(input: {
  url: string;
  incognito: boolean;
  ignored: boolean;
  hasPermission: boolean;
}): string | undefined {
  if (
    input.incognito ||
    input.ignored ||
    isRestrictedChromePage(input.url) ||
    (!input.url.startsWith("http://") && !input.url.startsWith("https://"))
  ) {
    return "CONTENT_NOT_ACCESSIBLE";
  }
  if (!input.hasPermission) return "CONTENT_PERMISSION_REQUIRED";
  return undefined;
}

function isRestrictedChromePage(input: string): boolean {
  try {
    const url = new URL(input);
    return (
      url.hostname === "chromewebstore.google.com" ||
      (url.hostname === "chrome.google.com" && url.pathname.startsWith("/webstore"))
    );
  } catch {
    return false;
  }
}

export function mapExtractionError(_error: unknown): {
  code: string;
  message: string;
} {
  return {
    code: "CONTENT_EXTRACTION_FAILED",
    message: "Visible text extraction failed",
  };
}

export async function sha256Hex(text: string): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(text));
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

export function createLineDiff(
  beforeText: string,
  afterText: string,
  maxCharacters = defaultDiffOutputLimit,
): LineDiff {
  if (
    !Number.isInteger(maxCharacters) ||
    maxCharacters < 1 ||
    maxCharacters > maximumDiffOutputLimit
  ) {
    throw new Error("maxCharacters must be between 1 and 50000");
  }

  const before = splitLines(beforeText);
  const after = splitLines(afterText);
  let prefix = 0;
  while (prefix < before.length && prefix < after.length && before[prefix] === after[prefix]) {
    prefix++;
  }
  let beforeSuffix = before.length;
  let afterSuffix = after.length;
  while (
    beforeSuffix > prefix &&
    afterSuffix > prefix &&
    before[beforeSuffix - 1] === after[afterSuffix - 1]
  ) {
    beforeSuffix--;
    afterSuffix--;
  }

  const beforeMiddle = before.slice(prefix, beforeSuffix);
  const afterMiddle = after.slice(prefix, afterSuffix);
  const canComputeMinimal = beforeMiddle.length * afterMiddle.length <= maximumLCSCells;
  const changes = canComputeMinimal
    ? createMinimalChanges(beforeMiddle, afterMiddle, prefix)
    : createReplacementChanges(beforeMiddle, afterMiddle, prefix);

  const originalCharacterCount = changes.reduce((total, change) => total + change.text.length, 0);
  const returned: LineChange[] = [];
  let returnedCharacterCount = 0;
  for (const change of changes) {
    if (returned.length >= maximumDiffEntries) break;
    const remaining = maxCharacters - returnedCharacterCount;
    if (change.text.length > remaining) {
      if (remaining > 0) {
        returned.push({ ...change, text: change.text.slice(0, remaining), textTruncated: true });
        returnedCharacterCount += remaining;
      }
      break;
    }
    returned.push(change);
    returnedCharacterCount += change.text.length;
  }

  return {
    changes: returned,
    minimal: canComputeMinimal,
    truncated:
      returned.length !== changes.length || returnedCharacterCount !== originalCharacterCount,
    originalChangeCount: changes.length,
    returnedChangeCount: returned.length,
    originalCharacterCount,
    returnedCharacterCount,
  };
}

function splitLines(text: string): string[] {
  return text === "" ? [] : text.split("\n");
}

function createMinimalChanges(
  before: string[],
  after: string[],
  lineOffset: number,
): LineChange[] {
  const rows = Array.from(
    { length: before.length + 1 },
    () => new Uint32Array(after.length + 1),
  );
  for (let beforeIndex = before.length - 1; beforeIndex >= 0; beforeIndex--) {
    for (let afterIndex = after.length - 1; afterIndex >= 0; afterIndex--) {
      rows[beforeIndex][afterIndex] =
        before[beforeIndex] === after[afterIndex]
          ? rows[beforeIndex + 1][afterIndex + 1] + 1
          : Math.max(rows[beforeIndex + 1][afterIndex], rows[beforeIndex][afterIndex + 1]);
    }
  }

  const changes: LineChange[] = [];
  let beforeIndex = 0;
  let afterIndex = 0;
  while (beforeIndex < before.length || afterIndex < after.length) {
    if (
      beforeIndex < before.length &&
      afterIndex < after.length &&
      before[beforeIndex] === after[afterIndex]
    ) {
      beforeIndex++;
      afterIndex++;
      continue;
    }
    if (
      afterIndex < after.length &&
      (beforeIndex === before.length ||
        rows[beforeIndex][afterIndex + 1] > rows[beforeIndex + 1][afterIndex])
    ) {
      changes.push({
        kind: "insert",
        newLine: lineOffset + afterIndex + 1,
        text: after[afterIndex],
      });
      afterIndex++;
      continue;
    }
    changes.push({
      kind: "delete",
      oldLine: lineOffset + beforeIndex + 1,
      text: before[beforeIndex],
    });
    beforeIndex++;
  }
  return changes;
}

function createReplacementChanges(
  before: string[],
  after: string[],
  lineOffset: number,
): LineChange[] {
  return [
    ...before.map((text, index): LineChange => ({
      kind: "delete",
      oldLine: lineOffset + index + 1,
      text,
    })),
    ...after.map((text, index): LineChange => ({
      kind: "insert",
      newLine: lineOffset + index + 1,
      text,
    })),
  ];
}

export function validateStableIdentity(
  before: ContentIdentity,
  after: ContentIdentity,
): void {
  if (
    before.tabId !== after.tabId ||
    before.url !== after.url ||
    before.documentId !== after.documentId
  ) {
    throw new Error("CONTENT_STALE");
  }
}

export class ContentRevisionStore {
  private readonly revisions = new Map<string, ContentIdentity>();

  constructor(
    private readonly randomID: () => string = () => crypto.randomUUID(),
    initial: Record<string, ContentIdentity> = {},
    private readonly onChange?: (value: Record<string, ContentIdentity>) => void,
  ) {
    for (const [revision, identity] of Object.entries(initial)) {
      this.revisions.set(revision, { ...identity });
    }
  }

  static async load(
    storage: {
      get(key: string): Promise<Record<string, unknown>>;
      set(value: Record<string, unknown>): Promise<void>;
    },
    randomID: () => string = () => crypto.randomUUID(),
  ): Promise<ContentRevisionStore> {
    const key = "contentRevisions";
    const stored = await storage.get(key);
    const initial = isIdentityRecord(stored[key]) ? stored[key] : {};
    return new ContentRevisionStore(randomID, initial, (value) => {
      void storage.set({ [key]: value });
    });
  }

  issue(identity: ContentIdentity): string {
    const revision = this.randomID();
    this.revisions.set(revision, { ...identity });
    this.persist();
    return revision;
  }

  isValid(revision: string, identity: ContentIdentity): boolean {
    const stored = this.revisions.get(revision);
    return (
      stored !== undefined &&
      stored.tabId === identity.tabId &&
      stored.url === identity.url &&
      stored.documentId === identity.documentId
    );
  }

  identity(revision: string): ContentIdentity | undefined {
    const stored = this.revisions.get(revision);
    return stored === undefined ? undefined : { ...stored };
  }

  invalidateTab(tabId: number): void {
    let changed = false;
    for (const [revision, identity] of this.revisions) {
      if (identity.tabId === tabId) {
        this.revisions.delete(revision);
        changed = true;
      }
    }
    if (changed) this.persist();
  }

  private persist(): void {
    if (this.onChange === undefined) return;
    this.onChange(Object.fromEntries(this.revisions));
  }
}

function isIdentityRecord(value: unknown): value is Record<string, ContentIdentity> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) return false;
  return Object.values(value).every(
    (identity) =>
      typeof identity === "object" &&
      identity !== null &&
      typeof (identity as ContentIdentity).tabId === "number" &&
      typeof (identity as ContentIdentity).url === "string" &&
      typeof (identity as ContentIdentity).documentId === "string",
  );
}

export function invalidateContentRevision(
  store: ContentRevisionStore,
  event: ContentInvalidationEvent,
): void {
  store.invalidateTab(event.tabId);
}

// This function is serialized by chrome.scripting. Keep it self-contained and
// limited to the main frame's visible text and current URL.
export function fixedPageExtractor(maxCharacters: number): {
  url: string;
  extracted: ExtractedText;
} {
  if (!Number.isInteger(maxCharacters) || maxCharacters < 1 || maxCharacters > 50_000) {
    throw new Error("invalid character limit");
  }
  const original = document.body?.innerText ?? "";
  const text = original.slice(0, maxCharacters);
  return {
    url: location.href,
    extracted: {
      text,
      truncated: original.length > maxCharacters,
      originalCharacterCount: original.length,
      returnedCharacterCount: text.length,
    },
  };
}

export function fixedIdentityProbe(): { url: string } {
  return { url: location.href };
}

// This function is serialized by chrome.scripting. Keep it self-contained and
// return only the digest and size, never the page text.
export async function fixedPageHasher(): Promise<PageHash> {
  const text = document.body?.innerText ?? "";
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(text));
  const sha256 = Array.from(
    new Uint8Array(digest),
    (byte) => byte.toString(16).padStart(2, "0"),
  ).join("");
  return { url: location.href, sha256, characterCount: text.length };
}

// This function is serialized by chrome.scripting. It returns bounded text to
// the extension service worker, where only changed lines are retained in the
// Native Messaging response.
export async function fixedPageDiffSnapshot(maxCharacters: number): Promise<PageDiffSnapshot> {
  if (!Number.isInteger(maxCharacters) || maxCharacters < 1 || maxCharacters > 50_000) {
    throw new Error("invalid character limit");
  }
  const original = document.body?.innerText ?? "";
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(original));
  const sha256 = Array.from(
    new Uint8Array(digest),
    (byte) => byte.toString(16).padStart(2, "0"),
  ).join("");
  const text = original.slice(0, maxCharacters);
  return {
    url: location.href,
    sha256,
    characterCount: original.length,
    text,
    truncated: original.length > maxCharacters,
    returnedCharacterCount: text.length,
  };
}
