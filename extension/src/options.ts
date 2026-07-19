import { normalizeTabIgnoreList } from "./settings";

const tabignoreInput = requiredElement<HTMLTextAreaElement>("tabignore");
const status = requiredElement<HTMLOutputElement>("status");

void chrome.storage.local.get("tabignore").then((stored) => {
  if (Array.isArray(stored.tabignore)) {
    tabignoreInput.value = stored.tabignore.filter((value) => typeof value === "string").join("\n");
  }
});

requiredElement<HTMLButtonElement>("save-tabignore").addEventListener("click", () => {
  void perform(async () => {
    const tabignore = normalizeTabIgnoreList(tabignoreInput.value);
    await chrome.storage.local.set({ tabignore });
    tabignoreInput.value = tabignore.join("\n");
    return "Ignored-tab patterns saved.";
  });
});

async function perform(action: () => Promise<string>): Promise<void> {
  try {
    status.textContent = await action();
  } catch (error) {
    status.textContent = error instanceof Error ? error.message : "Settings update failed.";
  }
}

function requiredElement<T extends HTMLElement>(id: string): T {
  const element = document.getElementById(id);
  if (element === null) throw new Error(`Missing settings element: ${id}`);
  return element as T;
}
