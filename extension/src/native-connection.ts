export interface NativePort {
  onDisconnect: {
    addListener(listener: () => void): void;
  };
  disconnect(): void;
}

export interface NativeConnectionOptions {
  initialDelayMs: number;
  maxDelayMs: number;
  maxRetries: number;
}

export class NativeConnectionManager {
  private running = false;
  private retryCount = 0;
  private port: NativePort | undefined;
  private retryTimer: ReturnType<typeof setTimeout> | undefined;

  constructor(
    private readonly connect: () => NativePort,
    private readonly options: NativeConnectionOptions,
  ) {}

  start(): void {
    if (this.running) return;
    this.running = true;
    this.open();
  }

  stop(): void {
    this.running = false;
    if (this.retryTimer !== undefined) clearTimeout(this.retryTimer);
    this.retryTimer = undefined;
    const port = this.port;
    this.port = undefined;
    port?.disconnect();
  }

  private open(): void {
    if (!this.running || this.port !== undefined) return;
    const port = this.connect();
    this.port = port;
    port.onDisconnect.addListener(() => {
      if (this.port !== port) return;
      this.port = undefined;
      this.scheduleRetry();
    });
  }

  private scheduleRetry(): void {
    if (!this.running || this.retryCount >= this.options.maxRetries) return;
    const delay = Math.min(
      this.options.initialDelayMs * 2 ** this.retryCount,
      this.options.maxDelayMs,
    );
    this.retryCount += 1;
    this.retryTimer = setTimeout(() => {
      this.retryTimer = undefined;
      this.open();
    }, delay);
  }
}
