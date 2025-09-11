import type { Methods, Events } from "./types";
type MethodNames = keyof Methods;
type EventNames = keyof Events;

// Internal message types (without jsonrpc field)
type RequestMessage<T = any> = {
  id: string | number;
  method: MethodNames;
  params: T;
};

type NotificationMessage<T = any> = {
  method: MethodNames;
  params: T;
};

type ResponseMessage<T = any> = {
  id: string | number;
  result?: T;
  error?: {
    code: number;
    message: string;
    // Can be anything, Only used for logging/debugging
    data?: any;
  };
};

type EventMessage<T = any> = {
  event: EventNames;
  data: T;
};

type IncomingMessage = ResponseMessage | EventMessage;

// Client options
export interface JsonRpcClientOptions {
  url: string;
  clientId: string;
  reconnectDelay?: number;
  maxReconnectAttempts?: number;
  timeout?: number;
}

// Event handlers type
type EventHandlers<E extends Events> = {
  [K in keyof E]?: (data: E[K]) => void;
};

export class JsonRpcWebSocketClient<
  M extends Methods = Methods,
  E extends Events = Events
> {
  private ws: WebSocket | null = null;
  private url: string;
  private clientId: string;
  private reconnectDelay: number;
  private maxReconnectAttempts: number;
  private timeout: number;
  private reconnectAttempts = 0;
  private isConnecting = false;
  private isManualClose = false;

  // Request tracking
  private pendingRequests = new Map<
    string | number,
    {
      resolve: (value: any) => void;
      reject: (error: any) => void;
      timeout: NodeJS.Timeout;
    }
  >();
  private requestId = 0;

  // Event handlers
  private eventHandlers: EventHandlers<E> = {};
  private connectionHandlers: {
    onConnect?: () => void;
    onDisconnect?: () => void;
    onError?: (error: Event) => void;
  } = {};

  constructor(options: JsonRpcClientOptions) {
    this.url = options.url;
    this.clientId = options.clientId;
    this.reconnectDelay = options.reconnectDelay ?? 1000;
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 5;
    this.timeout = options.timeout ?? 30000;
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
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    setTimeout(() => {
      if (!this.isManualClose) {
        this.connect().catch(() => {
          // Reconnection failed, will retry if attempts remaining
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

    if (message.error) {
      pending.reject(new Error(message.error.message));
    } else {
      pending.resolve(message.result);
    }
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
  call<K extends keyof M>(
    method: K,
    ...args: M[K]["res"] extends undefined
      ? [params: M[K]["req"]]
      : M[K]["req"] extends undefined
      ? []
      : [params: M[K]["req"]]
  ): M[K]["res"] extends undefined ? void : Promise<M[K]["res"]> {
    const params = args[0];

    // Handle notifications (no response expected)
    if (
      !("res" in (this as any).constructor.prototype) ||
      !(method as string).includes(".")
    ) {
      const message: NotificationMessage = {
        method: method as string,
        params,
      };

      this.send(message);
      return undefined as any;
    }

    // Handle regular method calls
    const id = ++this.requestId;
    const message: RequestMessage = {
      id,
      method: method as string,
      params,
    };

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(id);
        reject(new Error(`Request timeout for method: ${method as string}`));
      }, this.timeout);

      this.pendingRequests.set(id, { resolve, reject, timeout });
      this.send(message);
    }) as any;
  }

  // Type-safe method for notifications only
  notify<K extends keyof M>(
    method: K extends keyof M
      ? M[K]["res"] extends undefined
        ? K
        : never
      : never,
    params: M[K]["req"]
  ): void {
    const message: NotificationMessage = {
      method: method as string,
      params,
    };

    this.send(message);
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

// Usage example:
/*
// Define your types
type MyMethods = {
  "user.get": {
    req: { id: string };
    res: { id: string; name: string; email: string };
  };
  "user.delete": {
    req: { id: string };
    // No res - this is a notification
  };
  "ping": {
    req: undefined;
    res: { timestamp: number };
  };
};

type MyEvents = {
  "user.created": { id: string; name: string };
  "user.updated": { id: string; changes: Record<string, any> };
};

// Create client
const client = new JsonRpcWebSocketClient<MyMethods, MyEvents>({
  url: 'ws://localhost:8080',
  clientId: 'my-client-123',
  reconnectDelay: 1000,
  maxReconnectAttempts: 5,
  timeout: 30000
});

// Connect and use
await client.connect();

// Method call (returns promise)
const user = await client.call("user.get", { id: "123" });

// Notification (no response)
client.notify("user.delete", { id: "123" });

// Ping (no params)
const pong = await client.call("ping");

// Event handling
client.on("user.created", (data) => {
  console.log("User created:", data);
});

// Connection events
client.onConnect(() => console.log("Connected"));
client.onDisconnect(() => console.log("Disconnected"));
client.onError((error) => console.error("Error:", error));
*/
