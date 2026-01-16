import type { APIEvents, EventKind } from "./events";
import type { APIMethods, MethodKind } from "./methods";
import type { EventHandler, EventMessage, IncomingMessage, RequestMessage, ResponseMessage } from "./types";

// JSON-RPC error codes (https://www.jsonrpc.org/specification#error_object)
const RPC_ERROR_CODE = {
    PARSE_ERROR: -32700,
    INVALID_REQUEST: -32600,
    METHOD_NOT_FOUND: -32601,
    INVALID_PARAMS: -32602,
    INTERNAL_ERROR: -32603,
} as const;

// Client options
export interface WebSocketClientOptions {
    url: string;
    clientId: string;
    reconnectDelay?: number;
    maxReconnectAttempts?: number;
    requestTimeout?: number;
    connectionTimeout?: number;
    jsonReplacer?: (key: string, value: unknown) => unknown;
    jsonReviver?: (key: string, value: unknown) => unknown;
    onMessageParseError?: (error: Error, rawData: string) => void;
    logger?: (level: LogLevel, message: string) => void;
}

// Derived helper types
type PendingRequest = {
    resolve: (value: ResponseMessage<unknown>) => void;
    reject: (error: unknown) => void;
    timeout: ReturnType<typeof setTimeout>;
};

const loggers = {
    debug: console.debug,
    info: console.info,
    warn: console.warn,
    error: console.error,
} as const;
type LogLevel = keyof typeof loggers;

// Client
export class WebSocketClient {
    private ws: WebSocket | null = null;
    private url: string;
    private clientId: string;
    private reconnectDelay: number;
    private maxReconnectAttempts: number;
    private requestTimeout: number;
    private connectionTimeout: number;
    private reconnectAttempts = 0;
    private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    private connectionTimer: ReturnType<typeof setTimeout> | null = null;
    private isConnecting = false;
    private isManualClose = false;
    private jsonReplacer?: (key: string, value: unknown) => unknown;
    private jsonReviver?: (key: string, value: unknown) => unknown;
    private onMessageParseError?: (error: Error, rawData: string) => void;
    private logger: (level: LogLevel, message: string) => void;

    // Request tracking
    private pendingRequests = new Map<string, PendingRequest>();

    // Event handlers - multiple handlers per event
    private eventHandlers = new Map<EventKind, Set<EventHandler<APIEvents[EventKind]>>>();
    // Track events we've subscribed to on the server (separate from local handlers)
    private serverSubscriptions = new Set<EventKind>();
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
        this.requestTimeout = options.requestTimeout ?? 20000;
        this.connectionTimeout = options.connectionTimeout ?? 3000;
        this.jsonReplacer = options.jsonReplacer;
        this.jsonReviver = options.jsonReviver;
        this.onMessageParseError = options.onMessageParseError;
        this.logger = options.logger ?? this.defaultLogger.bind(this);
    }

    private defaultLogger(level: LogLevel, message: string): void {
        loggers[level](`[WebSocketClient] ${message}`);
    }

    // Connection timeout management
    private setupConnectionTimeout(reject: (error: Error) => void): void {
        this.connectionTimer = setTimeout(() => {
            if (this.ws?.readyState !== WebSocket.OPEN) {
                const message = `Connection timeout after ${this.connectionTimeout}ms`;
                this.ws?.close();
                this.isConnecting = false;
                this.connectionTimer = null;

                this.logger("warn", message);
                reject(new Error(message));
            }
        }, this.connectionTimeout);
    }

    private clearConnectionTimeout(): void {
        if (this.connectionTimer) {
            clearTimeout(this.connectionTimer);
            this.connectionTimer = null;
        }
    }

    private setupWebSocketHandlers(resolve: () => void, reject: (error: Error) => void): void {
        if (!this.ws) return;

        this.ws.onopen = () => {
            this.clearConnectionTimeout();
            this.isConnecting = false;
            this.reconnectAttempts = 0;
            this.logger("info", "Connection established");
            this.connectionHandlers.onConnect?.();

            // Resubscribe to all events (async, don't block connection)
            this.resubscribeAll().catch((error) => {
                this.logger("error", `Error during resubscription: ${error}`);
            });

            resolve();
        };

        this.ws.onmessage = (event) => {
            this.handleMessage(event.data);
        };

        this.ws.onclose = () => {
            this.clearConnectionTimeout();
            this.isConnecting = false;
            this.logger("info", "Connection closed");
            this.connectionHandlers.onDisconnect?.();

            if (!this.isManualClose) {
                this.scheduleReconnect();
            }
        };

        this.ws.onerror = (error) => {
            this.clearConnectionTimeout();
            this.isConnecting = false;
            this.logger("error", `Error occurred: ${error.type}`);
            this.connectionHandlers.onError?.(error);
            reject(new Error("WebSocket connection failed"));
        };
    }

    // Connection management
    async connect(): Promise<void> {
        if (this.isConnecting) return;
        if (this.ws?.readyState === WebSocket.OPEN) return;
        if (this.ws?.readyState === WebSocket.CONNECTING) return;

        this.isConnecting = true;
        this.isManualClose = false;

        return new Promise((resolve, reject) => {
            try {
                const newUrl = new URL(this.url);
                newUrl.searchParams.set("clientID", this.clientId);
                this.logger("info", `Connecting to URL: ${newUrl.toString()}`);
                this.ws = new WebSocket(newUrl.toString());

                this.setupConnectionTimeout(reject);
                this.setupWebSocketHandlers(resolve, reject);
            } catch (error) {
                this.clearConnectionTimeout();
                this.isConnecting = false;
                this.logger("error", `Connection error: ${error}`);
                reject(error);
            }
        });
    }

    disconnect(): void {
        this.logger("info", "Disconnecting client");
        this.isManualClose = true;
        this.reconnectAttempts = this.maxReconnectAttempts;

        // Clear reconnect timer if scheduled
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }

        // Clear connection timer if active
        if (this.connectionTimer) {
            clearTimeout(this.connectionTimer);
            this.connectionTimer = null;
        }

        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }

        // Reject all pending requests
        const pendingCount = this.pendingRequests.size;
        if (pendingCount > 0) {
            this.logger("warn", `Rejecting ${pendingCount} pending requests`);
        }
        for (const pending of this.pendingRequests.values()) {
            clearTimeout(pending.timeout);
            pending.reject(new Error("Connection closed"));
        }
        this.pendingRequests.clear();
    }

    private scheduleReconnect(): void {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            this.logger("warn", `Max reconnect attempts (${this.maxReconnectAttempts}) reached. Giving up.`);
            return;
        }

        this.reconnectAttempts++;

        // Run user-defined reconnect attempt handler
        this.connectionHandlers.onReconnectAttempt?.(this.reconnectAttempts);

        // Calculate delay with exponential backoff, but cap it at a reasonable maximum
        const maxDelay = 10000; // Cap at 10 seconds
        // Cap exponent at 10 to prevent overflow (2^10 = 1024)
        const exponent = Math.min(this.reconnectAttempts - 1, 10);
        const baseDelay = this.reconnectDelay * 2 ** exponent;
        const delay = Math.min(baseDelay, maxDelay);

        const msg = `Scheduling reconnect attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`;
        this.logger("info", msg);

        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            if (!this.isManualClose) {
                this.connect().catch(() => {
                    // Reconnection failed, will retry if attempts remaining
                    this.logger("warn", `Reconnect attempt ${this.reconnectAttempts} failed`);
                });
            }
        }, delay);
    }

    private handleEvent(message: EventMessage) {
        const handlers = this.eventHandlers.get(message.event);
        if (!handlers || handlers.size === 0) {
            this.logger("warn", `No handler registered for event: ${String(message.event)}`);
            return;
        }
        // Call all handlers for this event
        for (const handler of handlers) {
            try {
                handler(message.data);
            } catch (error) {
                this.logger("error", `Error in event handler for ${String(message.event)}: ${error}`);
            }
        }
    }

    private handleResponse(message: ResponseMessage) {
        const pending = this.pendingRequests.get(message.id);
        if (!pending) {
            this.logger("warn", `No pending request found for id: ${message.id}`);
            return;
        }
        this.pendingRequests.delete(message.id);
        clearTimeout(pending.timeout);

        pending.resolve(message);
    }

    // Message handling
    private handleMessage(data: string): void {
        try {
            const message: IncomingMessage = JSON.parse(data, this.jsonReviver);

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
            this.logger("warn", `Received invalid message: ${JSON.stringify(message)}`);
        } catch (error) {
            const err = error instanceof Error ? error : new Error(String(error));
            this.logger("error", `Failed to parse message: ${err}`);
            this.onMessageParseError?.(err, data);
        }
    }

    private send(message: unknown): void {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            const error = new Error("WebSocket is not connected");
            this.logger("error", `Failed to send message: ${error.message}`);
            throw error;
        }

        this.ws.send(JSON.stringify(message, this.jsonReplacer));
    }

    // Private implementation that accepts all methods
    private async _call(method: MethodKind, params?: unknown): Promise<ResponseMessage<unknown>> {
        // Handle regular method calls
        const id = crypto.randomUUID();
        const message: RequestMessage = {
            jsonrpc: "2.0",
            id,
            method,
            params: params as APIMethods[MethodKind]["req"],
        };

        function createErrorResponse(msg: string): ResponseMessage<never> {
            return {
                jsonrpc: "2.0",
                id,
                error: {
                    code: RPC_ERROR_CODE.INTERNAL_ERROR,
                    message: `[${String(method)}] ${msg}`,
                },
            };
        }

        return new Promise((resolve) => {
            const timeout = setTimeout(() => {
                this.pendingRequests.delete(id);
                const errorMsg = `Request timed out after ${this.requestTimeout}ms`;
                this.logger("warn", `[${String(method)}] ${errorMsg}`);
                resolve(createErrorResponse(errorMsg));
            }, this.requestTimeout);

            this.pendingRequests.set(id, {
                resolve,
                reject: () => {
                    resolve(createErrorResponse("Connection closed"));
                },
                timeout,
            });

            try {
                this.send(message);
            } catch (error) {
                clearTimeout(timeout);
                this.pendingRequests.delete(id);
                const msg = error instanceof Error ? error.message : "Failed to send message";
                this.logger("error", `[${String(method)}] ${msg}`);
                resolve(createErrorResponse(msg));
            }
        });
    }

    // Public API methods
    // Exclude subscribe/unsubscribe as they have dedicated methods
    // For methods without parameters (where req is never)
    async call<M extends Exclude<MethodKind, "subscribe" | "unsubscribe">>(
        method: APIMethods[M]["req"] extends never ? M : never
    ): Promise<ResponseMessage<APIMethods[M]["res"]>>;

    // For methods with parameters
    async call<M extends Exclude<MethodKind, "subscribe" | "unsubscribe">>(
        method: M,
        params: APIMethods[M]["req"]
    ): Promise<ResponseMessage<APIMethods[M]["res"]>>;

    /**
     * Call a method on the server.
     */
    async call(method: MethodKind, params?: unknown): Promise<ResponseMessage<unknown>> {
        return this._call(method, params);
    }

    /**
     * Add an event handler.
     * Will automatically resubscribe if the connection is lost.
     */
    on<E extends EventKind>(event: E, handler: EventHandler<APIEvents[E]>): void {
        let handlers = this.eventHandlers.get(event);
        if (!handlers) {
            handlers = new Set();
            this.eventHandlers.set(event, handlers);
        }
        // Cast needed due to TypeScript variance with Set
        handlers.add(handler as EventHandler<APIEvents[EventKind]>);
    }

    /**
     * Remove an event handler.
     */
    off<E extends EventKind>(event: E, handler: EventHandler<APIEvents[E]>): void {
        const handlers = this.eventHandlers.get(event);
        if (!handlers) return;

        if (!handlers.has(handler as EventHandler<APIEvents[EventKind]>)) {
            const reasons = [
                "You passed an inline function instead of a bound function, ie `() => {}` vs `myFunction`",
                "You called off() twice with the same handler",
                "You never registered the handler with on()",
            ];
            this.logger("warn", `Handler not found for event: ${String(event)}. Reasons: ${reasons.join("\n")}`);
            return;
        }

        // Cast needed due to TypeScript variance with Set
        handlers.delete(handler as EventHandler<APIEvents[EventKind]>);
        // If no more handlers for this event, remove the entry
        if (handlers.size === 0) this.eventHandlers.delete(event);
    }

    /**
     * Subscribe to an event.
     * Will automatically resubscribe if the connection is lost.
     */
    async subscribe(event: EventKind): Promise<ResponseMessage<APIMethods["subscribe"]["res"]>> {
        // Track this subscription
        this.serverSubscriptions.add(event);

        // Call subscribe on the server
        const response = (await this._call("subscribe", { event })) as ResponseMessage<APIMethods["subscribe"]["res"]>;

        if (response.error) {
            this.logger("error", `Failed to subscribe to ${String(event)}: ${response.error.message}`);
        } else {
            this.logger("debug", `Subscribed to event: ${String(event)}`);
        }

        return response;
    }

    /**
     * Unsubscribe from an event.
     * Will also remove any handlers for the event.
     */
    async unsubscribe(event: EventKind): Promise<ResponseMessage<APIMethods["unsubscribe"]["res"]>> {
        // Remove from tracked subscriptions
        this.serverSubscriptions.delete(event);
        this.eventHandlers.delete(event);

        // Call unsubscribe on the server
        const response = (await this._call("unsubscribe", { event })) as ResponseMessage<
            APIMethods["unsubscribe"]["res"]
        >;

        if (response.error) {
            this.logger("error", `Failed to unsubscribe from ${String(event)}: ${response.error.message}`);
        } else {
            this.logger("debug", `Unsubscribed from event: ${String(event)}`);
        }

        return response;
    }

    private async resubscribeAll(): Promise<void> {
        if (this.serverSubscriptions.size === 0) return;

        this.logger("info", `Resubscribing to ${this.serverSubscriptions.size} event(s)`);

        // Resubscribe to all tracked events
        const subscriptions = Array.from(this.serverSubscriptions).map(async (event) => {
            try {
                const response = await this._call("subscribe", { event });

                if (response.error) {
                    this.logger("warn", `Failed to resubscribe to ${String(event)}: ${response.error.message}`);
                } else {
                    this.logger("debug", `Resubscribed to event: ${String(event)}`);
                }
            } catch (error) {
                this.logger("error", `Error resubscribing to ${String(event)}: ${error}`);
            }
        });

        await Promise.allSettled(subscriptions);
    }

    /**
     * Called when the connection is established.
     */
    onConnect(handler: () => void): void {
        this.connectionHandlers.onConnect = handler;
    }

    /**
     * Called when the connection is lost.
     */
    onDisconnect(handler: () => void): void {
        this.connectionHandlers.onDisconnect = handler;
    }

    /**
     * Called when an error occurs.
     */
    onError(handler: (error: Event) => void): void {
        this.connectionHandlers.onError = handler;
    }

    /**
     * Called when a reconnect attempt is made.
     */
    onReconnectAttempt(handler: (attempt: number) => void): void {
        this.connectionHandlers.onReconnectAttempt = handler;
    }

    /**
     * Check if the client is connected to the server.
     */
    get isConnected(): boolean {
        return this.ws?.readyState === WebSocket.OPEN;
    }

    /**
     * Get the current connection state.
     */
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
