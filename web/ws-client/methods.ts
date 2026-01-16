import type { EventKind } from "./events";

export type MethodKind = keyof APIMethods;
/**
 * Mapping of API methods to their request and response types
 * Use `never` for methods without parameters
 * */
export type APIMethods = {
    ping: { req: never; res: PingResult };
    subscribe: { req: SubscribeParams; res: SubscribeResult };
    unsubscribe: { req: UnsubscribeParams; res: UnsubscribeResult };
};

export type SubscribeParams = {
    /** The event topic to subscribe to */
    event: EventKind;
};

export type SubscribeResult = {
    /** Whether the subscribe was successful */
    success: boolean;
};

export type UnsubscribeParams = {
    /** The event topic to unsubscribe from */
    event: EventKind;
};

export type UnsubscribeResult = {
    /** Whether the unsubscribe was successful */
    success: boolean;
};

export type PingStatus = "success" | "error";
export type PingResult = {
    /** A message describing the result */
    message: string;
    /** The status of the ping */
    status: PingStatus;
};
