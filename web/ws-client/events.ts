import type * as T from "./generated";
export type EventKind = keyof APIEvents;
/**
 * Mapping of event names to their data types
 */
export type APIEvents = {
    "data.created": T.DataCreated;
};
