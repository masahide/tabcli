import { afterEach, describe, expect, it, vi } from "vitest";

import {
  NativeConnectionManager,
  type NativePort,
} from "../src/native-connection";

class FakePort implements NativePort {
  private listeners: Array<() => void> = [];
  disconnected = false;

  readonly onDisconnect = {
    addListener: (listener: () => void) => this.listeners.push(listener),
  };

  disconnect(): void {
    if (this.disconnected) return;
    this.disconnected = true;
    for (const listener of this.listeners) listener();
  }
}

afterEach(() => {
  vi.useRealTimers();
});

describe("NativeConnectionManager", () => {
  it("uses bounded exponential backoff and keeps only one connection", async () => {
    vi.useFakeTimers();
    const ports: FakePort[] = [];
    let activeConnections = 0;
    let maxConcurrent = 0;

    const connect = vi.fn(() => {
      expect(activeConnections).toBe(0);
      const port = new FakePort();
      const originalDisconnect = port.disconnect.bind(port);
      port.disconnect = () => {
        if (!port.disconnected) activeConnections -= 1;
        originalDisconnect();
      };
      ports.push(port);
      activeConnections += 1;
      maxConcurrent = Math.max(maxConcurrent, activeConnections);
      return port;
    });

    const manager = new NativeConnectionManager(connect, {
      initialDelayMs: 100,
      maxDelayMs: 400,
      maxRetries: 3,
    });

    manager.start();
    manager.start();
    expect(connect).toHaveBeenCalledTimes(1);

    ports[0].disconnect();
    await vi.advanceTimersByTimeAsync(99);
    expect(connect).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(connect).toHaveBeenCalledTimes(2);

    ports[1].disconnect();
    await vi.advanceTimersByTimeAsync(200);
    ports[2].disconnect();
    await vi.advanceTimersByTimeAsync(400);
    expect(connect).toHaveBeenCalledTimes(4);

    ports[3].disconnect();
    await vi.advanceTimersByTimeAsync(10_000);
    expect(connect).toHaveBeenCalledTimes(4);
    expect(maxConcurrent).toBe(1);
  });

  it("stop cancels a pending retry", async () => {
    vi.useFakeTimers();
    const firstPort = new FakePort();
    const connect = vi.fn(() => firstPort);
    const manager = new NativeConnectionManager(connect, {
      initialDelayMs: 100,
      maxDelayMs: 400,
      maxRetries: 3,
    });

    manager.start();
    firstPort.disconnect();
    manager.stop();
    await vi.runAllTimersAsync();

    expect(connect).toHaveBeenCalledTimes(1);
  });
});
