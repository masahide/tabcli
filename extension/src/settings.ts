import { normalizeTabIgnorePattern } from "./snapshot";

export function originPermissionPattern(input: string): string {
  const url = new URL(input);
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    throw new Error("Only HTTP and HTTPS origins are supported");
  }
  return `${url.protocol}//${url.hostname}/*`;
}

export function normalizeTabIgnoreList(value: string): string[] {
  return value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line !== "")
    .map(normalizeTabIgnorePattern);
}
