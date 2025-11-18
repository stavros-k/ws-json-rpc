// A map of event data indexed by event ID
export type EventDataMap = Record<string, SomeEvent>;

// All the available event topics
export const EventKind = {
    /** Data created */
    DataCreated: "data.created",
    /** Data updated */
    DataUpdated: "data.updated",
} as const;

export type EventKind = (typeof EventKind)[keyof typeof EventKind];

export function isEventKind(value: unknown): value is EventKind {
    switch (value) {
        case "data.created":
        case "data.updated":
            return true;
        default:
            return false;
    }
}

// All the available RPC methods
export const MethodKind = {
    /** Ping */
    Ping: "ping",
    /** Subscribe */
    Subscribe: "subscribe",
    /** Unsubscribe */
    Unsubscribe: "unsubscribe",
    /** Create user */
    UserCreate: "user.create",
    /** Update user */
    UserUpdate: "user.update",
    /** Delete user */
    UserDelete: "user.delete",
    /** List users */
    UserList: "user.list",
    /** Get user */
    UserGet: "user.get",
} as const;

export type MethodKind = (typeof MethodKind)[keyof typeof MethodKind];

export function isMethodKind(value: unknown): value is MethodKind {
    switch (value) {
        case "ping":
        case "subscribe":
        case "unsubscribe":
        case "user.create":
        case "user.update":
        case "user.delete":
        case "user.list":
        case "user.get":
            return true;
        default:
            return false;
    }
}

// Result for the Ping method
export type PingResult = {
    /** A message describing the result */
    message: string;
    /** The status of the ping */
    status: PingStatus;
};

// Status for the Ping method
export const PingStatus = {
    /** Success */
    Success: "success",
    /** Error */
    Error: "error",
} as const;

export type PingStatus = (typeof PingStatus)[keyof typeof PingStatus];

export function isPingStatus(value: unknown): value is PingStatus {
    switch (value) {
        case "success":
        case "error":
            return true;
        default:
            return false;
    }
}

// Result for the SomeEvent method
export type SomeEvent = {
    /** The unique identifier for the result */
    id: string;
};

// Result for the Status method
export type StatusResult = PingResult;

// A map with string values for storing key-value pairs
export type StringMap = Record<string, string>;

// Parameters for the Subscribe method
export type SubscribeParams = {
    /** The event topic to subscribe to */
    event: EventKind;
};

// Result for the Subscribe method
export type SubscribeResult = {
    /** Whether the subscribe was successful */
    success: boolean;
};

// Parameters for the Unsubscribe method
export type UnsubscribeParams = {
    /** The event topic to unsubscribe from */
    event: EventKind;
};

// Result for the Unsubscribe method
export type UnsubscribeResult = {
    /** Whether the unsubscribe was successful */
    success: boolean;
};

/**
 * Type mapping for RPC methods.
 * Maps method names to their request and response types.
 */
export type ApiMethods = {
    ping: { req: never; res: PingResult };
    subscribe: { req: SubscribeParams; res: SubscribeResult };
    unsubscribe: { req: UnsubscribeParams; res: UnsubscribeResult };
};

/**
 * Type mapping for WebSocket events.
 * Maps event names to their data types.
 */
export type ApiEvents = {
    "data.created": { data: SomeEvent };
};
