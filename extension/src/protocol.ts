export const protocolVersion = 3;

export interface NativeEnvelope<T = unknown> {
  protocolVersion: number;
  id: string;
  operation: string;
  payload?: T;
  error?: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
}

export function createHandshake(): NativeEnvelope<{ extensionVersion: string; profileId: string }> {
  return {
    protocolVersion,
    id: crypto.randomUUID(),
    operation: "handshake",
    payload: { extensionVersion: chrome.runtime.getManifest().version, profileId: "default" },
  };
}

export function validateEnvelope(value: unknown): asserts value is NativeEnvelope {
  if (
    typeof value !== "object" ||
    value === null ||
    (value as NativeEnvelope).protocolVersion !== protocolVersion ||
    typeof (value as NativeEnvelope).id !== "string" ||
    typeof (value as NativeEnvelope).operation !== "string"
  ) {
    throw new Error("PROTOCOL_VERSION_MISMATCH");
  }
}

export interface HostHandshake {
  accepted: boolean;
  hostVersion: string;
  profileId: string;
  minimumProtocolVersion: number;
  maximumProtocolVersion: number;
  updateInstructions: string;
}

export function validateHostHandshake(payload: unknown): asserts payload is HostHandshake {
  const value = payload as Partial<HostHandshake> | null;
  if (
    value === null ||
    typeof value !== "object" ||
    value.accepted !== true ||
    typeof value.hostVersion !== "string" ||
    value.profileId !== "default" ||
    typeof value.minimumProtocolVersion !== "number" ||
    typeof value.maximumProtocolVersion !== "number" ||
    protocolVersion < value.minimumProtocolVersion ||
    protocolVersion > value.maximumProtocolVersion ||
    typeof value.updateInstructions !== "string"
  ) {
    throw new Error("PROTOCOL_VERSION_MISMATCH: Update the extension and tabcli together, then restart Chrome.");
  }
}
