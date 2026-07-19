import { describe, expect, it } from "vitest";
import manifest from "../manifest.json";

describe("extension manifest", () => {
  it("declares the permissions required by the Native Messaging architecture", () => {
    expect(manifest.permissions).toContain("nativeMessaging");
  });

  it("declares all HTTP and HTTPS sites as required host permissions", () => {
    expect(manifest.host_permissions).toEqual([
      "http://*/*",
      "https://*/*",
    ]);
    expect(manifest).not.toHaveProperty("optional_host_permissions");
    expect(manifest).not.toHaveProperty("content_scripts");
    expect(manifest.permissions).toContain("scripting");
    expect(manifest.permissions).not.toContain("activeTab");
  });
});
