import { protocolVersion } from "./protocol";

export interface TabsRemoveAPI {
  remove(tabIds: number | number[]): Promise<void>;
}

export class TabCloseError extends Error {
  readonly code: "INVALID_ARGUMENT" | "TAB_CLOSE_FAILED";
  readonly details?: Record<string, unknown>;

  constructor(
    code: "INVALID_ARGUMENT" | "TAB_CLOSE_FAILED",
    message: string,
    details?: Record<string, unknown>,
  ) {
    super(message);
    this.code = code;
    this.details = details;
  }
}

export async function closeTabs(
  tabs: TabsRemoveAPI,
  input: { tabIds: number[] },
): Promise<{ protocolVersion: number; closedTabIds: number[] }> {
  if (
    !Array.isArray(input.tabIds) ||
    input.tabIds.length === 0 ||
    input.tabIds.some((tabId) => !Number.isInteger(tabId) || tabId <= 0) ||
    new Set(input.tabIds).size !== input.tabIds.length
  ) {
    throw new TabCloseError(
      "INVALID_ARGUMENT",
      "tabIds must contain unique positive integers",
    );
  }

  try {
    await tabs.remove(input.tabIds);
  } catch {
    throw new TabCloseError(
      "TAB_CLOSE_FAILED",
      "Chrome could not close all selected tabs",
      { tabIds: input.tabIds },
    );
  }
  return { protocolVersion, closedTabIds: [...input.tabIds] };
}
