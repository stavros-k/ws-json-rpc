import type { EventKind } from "./events";

export type MethodKind = keyof APIMethods;
export type APIMethods = {
    ping: { req: never; res: PingResult };
    subscribe: { req: SubscribeParams; res: SubscribeResult };
    unsubscribe: { req: UnsubscribeParams; res: UnsubscribeResult };
};

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

export type PingStatus = (typeof PingStatus)[keyof typeof PingStatus];
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
