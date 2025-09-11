export type Methods = {
  [method: string]: {
    req: any;
    res: any;
  };
};

export type Events = {
  [event: string]: any;
};

type MethodNames = keyof Methods;
type EventNames = keyof Events;

// Internal message types (without jsonrpc field)
type RequestMessage<T = any> = {
  id: ReturnType<typeof crypto.randomUUID>;
  method: MethodNames;
  params: T;
};

type SuccessResult<T> = {
  result: T;
  error?: never;
};

type ErrorResult = {
  result?: never;
  error: {
    code: number;
    message: string;
    // Can be anything, Only used for logging/debugging
    data?: any;
  };
};

type ResponseMessage<T = any> = {
  id: ReturnType<typeof crypto.randomUUID>;
} & (SuccessResult<T> | ErrorResult);

type EventMessage<T = any> = {
  event: EventNames;
  data: T;
};

type IncomingMessage = ResponseMessage | EventMessage;

// Event handlers type
type EventHandlers<E extends Events> = {
  [K in keyof E]?: (data: E[K]) => void;
};

// Client options
export interface JsonRpcClientOptions {
  url: string;
  clientId: string;
  reconnectDelay?: number;
  maxReconnectAttempts?: number;
  requestTimeout?: number;
}

export class JsonRpcWebSocketClient<
  M extends Methods = Methods,
  E extends Events = Events
> {
  private ws: WebSocket | null = null;
  private url: string;
  private clientId: string;
  private reconnectDelay: number;
  private maxReconnectAttempts: number;
  private requestTimeout: number;
  private reconnectAttempts = 0;
  private isConnecting = false;
  private isManualClose = false;

  // Request tracking
  private pendingRequests = new Map<
    string,
    {
      resolve: (value: ResponseMessage<any>) => void;
      reject: (error: any) => void;
      timeout: ReturnType<typeof setTimeout>;
    }
  >();

  // Event handlers
  private eventHandlers: EventHandlers<E> = {};
  private connectionHandlers: {
    onConnect?: () => void;
    onDisconnect?: () => void;
    onError?: (error: Event) => void;
    onReconnectAttempt?: (attempt: number) => void;
  } = {};

  constructor(options: JsonRpcClientOptions) {
    this.url = options.url;
    this.clientId = options.clientId;
    this.reconnectDelay = options.reconnectDelay ?? 1000;
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 5;
    this.requestTimeout = options.requestTimeout ?? 30000;
  }

  // Connection management
  async connect(): Promise<void> {
    if (
      this.isConnecting ||
      (this.ws && this.ws.readyState === WebSocket.OPEN)
    ) {
      return;
    }

    this.isConnecting = true;
    this.isManualClose = false;

    return new Promise((resolve, reject) => {
      try {
        const newUrl = new URL(this.url);
        newUrl.searchParams.set("clientID", this.clientId);
        this.ws = new WebSocket(newUrl.toString());

        this.ws.onopen = () => {
          this.isConnecting = false;
          this.reconnectAttempts = 0;
          this.connectionHandlers.onConnect?.();
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onclose = () => {
          this.isConnecting = false;
          this.connectionHandlers.onDisconnect?.();

          if (!this.isManualClose) {
            this.scheduleReconnect();
          }
        };

        this.ws.onerror = (error) => {
          this.isConnecting = false;
          this.connectionHandlers.onError?.(error);
          reject(new Error("WebSocket connection failed"));
        };
      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }

  disconnect(): void {
    this.isManualClose = true;
    this.reconnectAttempts = this.maxReconnectAttempts;

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    // Reject all pending requests
    for (const [id, pending] of this.pendingRequests) {
      clearTimeout(pending.timeout);
      pending.reject(new Error("Connection closed"));
    }
    this.pendingRequests.clear();
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.warn(
        `Max reconnect attempts (${this.maxReconnectAttempts}) reached. Giving up.`
      );
      return;
    }

    this.reconnectAttempts++;

    // Run user-defined reconnect attempt handler
    this.connectionHandlers.onReconnectAttempt?.(this.reconnectAttempts);

    // Calculate delay with exponential backoff, but cap it at a reasonable maximum
    const maxDelay = 30000; // Cap at 30 seconds
    const baseDelayMultiplier = Math.min(this.reconnectAttempts - 1, 10);
    const baseDelay = this.reconnectDelay * Math.pow(2, baseDelayMultiplier);
    const delay = Math.min(baseDelay, maxDelay);

    console.log(
      `Scheduling reconnect attempt ${this.reconnectAttempts} in ${delay}ms`
    );

    setTimeout(() => {
      if (!this.isManualClose) {
        this.connect().catch(() => {
          // Reconnection failed, will retry if attempts remaining
          console.warn(`Reconnect attempt ${this.reconnectAttempts} failed`);
        });
      }
    }, delay);
  }

  private handleEvent(message: EventMessage) {
    const handler = this.eventHandlers[message.event];
    if (!handler) {
      console.warn(`No handler registered for event: ${message.event}`);
      return;
    }
    handler(message.data);
  }

  private handleResponse(message: ResponseMessage) {
    const pending = this.pendingRequests.get(message.id);
    if (!pending) {
      console.warn(`No pending request found for id: ${message.id}`);
      return;
    }
    this.pendingRequests.delete(message.id);
    clearTimeout(pending.timeout);

    pending.resolve(message);
  }

  // Message handling
  private handleMessage(data: string): void {
    try {
      const message: IncomingMessage = JSON.parse(data);

      // Handle response
      if ("id" in message) {
        this.handleResponse(message);
        return;
      }

      // Handle event
      if ("event" in message) {
        this.handleEvent(message);
        return;
      }
    } catch (error) {
      console.error("Failed to parse message:", error);
    }
  }

  private send(message: any): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error("WebSocket is not connected");
    }

    this.ws.send(JSON.stringify(message));
  }

  // Public API methods
  async call<K extends keyof M>(
    method: K,
    ...args: M[K]["res"] extends undefined
      ? [params: M[K]["req"]]
      : M[K]["req"] extends undefined
      ? []
      : [params: M[K]["req"]]
  ): Promise<ResponseMessage<M[K]["res"]>> {
    const params = args[0];

    // Handle regular method calls
    const id = crypto.randomUUID();
    const message: RequestMessage = {
      id,
      method: method as MethodNames,
      params,
    };

    function newRejectObj(msg: string): ResponseMessage<any> {
      return {
        id,
        error: { code: 32603, message: `[${String(method)}] ${msg}` },
      };
    }

    return new Promise((resolve) => {
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(id);
        resolve(newRejectObj("Request timeout for method"));
      }, this.requestTimeout);

      this.pendingRequests.set(id, {
        resolve,
        reject: () => {
          resolve(newRejectObj("Connection closed"));
        },
        timeout,
      });

      try {
        this.send(message);
      } catch (error) {
        clearTimeout(timeout);
        this.pendingRequests.delete(id);
        const msg =
          error instanceof Error ? error.message : "Failed to send message";
        resolve(newRejectObj(msg));
      }
    });
  }

  // Event subscription
  on<K extends keyof E>(event: K, handler: (data: E[K]) => void): void {
    this.eventHandlers[event] = handler;
  }

  off<K extends keyof E>(event: K): void {
    delete this.eventHandlers[event];
  }

  // Connection event handlers
  onConnect(handler: () => void): void {
    this.connectionHandlers.onConnect = handler;
  }

  onDisconnect(handler: () => void): void {
    this.connectionHandlers.onDisconnect = handler;
  }

  onError(handler: (error: Event) => void): void {
    this.connectionHandlers.onError = handler;
  }

  onReconnectAttempt(handler: (attempt: number) => void): void {
    this.connectionHandlers.onReconnectAttempt = handler;
  }

  // Utility methods
  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  get connectionState(): "connecting" | "open" | "closing" | "closed" {
    if (!this.ws) return "closed";

    switch (this.ws.readyState) {
      case WebSocket.CONNECTING:
        return "connecting";
      case WebSocket.OPEN:
        return "open";
      case WebSocket.CLOSING:
        return "closing";
      case WebSocket.CLOSED:
        return "closed";
      default:
        return "closed";
    }
  }
}
