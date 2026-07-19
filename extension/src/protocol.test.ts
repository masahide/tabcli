import { describe, expect, it } from "vitest";
import { protocolVersion, validateHostHandshake } from "./protocol";

describe("host compatibility", () => {
  it("accepts a host range containing the extension protocol", () => {
    expect(() => validateHostHandshake({
      accepted: true,
      hostVersion: "0.1.0",
      profileId: "default",
      minimumProtocolVersion: protocolVersion,
      maximumProtocolVersion: protocolVersion,
      updateInstructions: "update together",
    })).not.toThrow();
  });

  it("rejects an incompatible host with update guidance", () => {
    expect(() => validateHostHandshake({
      accepted: true,
      hostVersion: "2.0.0",
      profileId: "default",
      minimumProtocolVersion: protocolVersion + 1,
      maximumProtocolVersion: protocolVersion + 1,
      updateInstructions: "update together",
    })).toThrow(/Update the extension and tabcli together/);
  });
});
