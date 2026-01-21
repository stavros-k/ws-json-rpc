import type * as T from "./generated";

export type MethodKind = keyof APIMethods;
/**
 * Mapping of API methods to their request and response types
 * Use `never` for methods without parameters
 * */
export type APIMethods = {
    ping: { req: never; res: T.PingResult };
    subscribe: { req: T.SubscribeParams; res: T.SubscribeResult };
    unsubscribe: { req: T.UnsubscribeParams; res: T.UnsubscribeResult };
};
