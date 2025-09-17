// Basic types
type UUID = ReturnType<typeof crypto.randomUUID>;

// Base constraint types
type MethodsRecord = Record<string, { req: any; res: any }>;
type EventsRecord = Record<string, any>;

// Request message sent to the server
type RequestMessage<
  Methods extends MethodsRecord,
  Method extends keyof Methods
> = {
  id: UUID;
  method: Method;
  params: Methods[Method]["req"];
};

// Response message with either result or error
type ResponseMessage<Result = any> = {
  id: UUID;
} & (
  | { result: Result; error?: never }
  | { result?: never; error: { code: number; message: string; data?: any } }
);

// Event message is an object containing the event name and its data
type EventMessage<Events extends EventsRecord, Event extends keyof Events> = {
  event: Event;
  data: Events[Event];
};

// Incoming message is either a response or an event
type IncomingMessage<Events extends EventsRecord> =
  | ResponseMessage
  | EventMessage<Events, keyof Events>;

// Event handlers map
type EventHandlers<Events extends EventsRecord> = {
  [Event in keyof Events]?: (data: Events[Event]) => void;
};

// Client options
export interface WebSocketClientOptions {
  url: string;
  clientId: string;
  reconnectDelay?: number;
  maxReconnectAttempts?: number;
  requestTimeout?: number;
  jsonReplacer?: (key: string, value: any) => any;
  jsonReviver?: (key: string, value: any) => any;
}

// Client
export class WebSocketClient<
  Methods extends MethodsRecord = MethodsRecord,
  Events extends EventsRecord = EventsRecord
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
  private jsonReplacer?: (key: string, value: any) => any;
  private jsonReviver?: (key: string, value: any) => any;

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
  private eventHandlers: EventHandlers<Events> = {};
  private connectionHandlers: {
    onConnect?: () => void;
    onDisconnect?: () => void;
    onError?: (error: Event) => void;
    onReconnectAttempt?: (attempt: number) => void;
  } = {};

  constructor(options: WebSocketClientOptions) {
    this.url = options.url;
    this.clientId = options.clientId;
    this.reconnectDelay = options.reconnectDelay ?? 1000;
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 5;
    this.requestTimeout = options.requestTimeout ?? 30000;
    this.jsonReplacer = options.jsonReplacer;
    this.jsonReviver = options.jsonReviver;
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

  private handleEvent(message: EventMessage<Events, keyof Events>) {
    const handler = this.eventHandlers[message.event];
    if (!handler) {
      console.warn(`No handler registered for event: ${String(message.event)}`);
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
      const message: IncomingMessage<Events> = JSON.parse(
        data,
        this.jsonReviver
      );

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

    this.ws.send(JSON.stringify(message, this.jsonReplacer));
  }

  // Public API methods
  // For methods without parameters
  async call<Method extends keyof Methods>(
    method: Methods[Method]["req"] extends undefined ? Method : never
  ): Promise<ResponseMessage<Methods[Method]["res"]>>;

  // For methods with parameters
  async call<Method extends keyof Methods>(
    method: Methods[Method]["req"] extends undefined ? never : Method,
    params: Methods[Method]["req"]
  ): Promise<ResponseMessage<Methods[Method]["res"]>>;

  // Implementation
  async call<Method extends keyof Methods>(
    method: Method,
    params?: Methods[Method]["req"]
  ): Promise<ResponseMessage<Methods[Method]["res"]>> {
    // Handle regular method calls
    const id = crypto.randomUUID();
    const message: RequestMessage<Methods, Method> = { id, method, params };

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
  on<Event extends keyof Events>(
    event: Event,
    handler: (data: Events[Event]) => void
  ): void {
    this.eventHandlers[event] = handler;
  }

  off<Event extends keyof Events>(event: Event): void {
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
