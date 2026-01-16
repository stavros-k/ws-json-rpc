export type EventKind = keyof APIEvents;
/**
 * Mapping of event names to their data types
 */
export type APIEvents = {
    "data.created": DataCreatedEvent;
    "data.deleted": DataDeletedEvent;
};

export type DataCreatedEvent = {
    data: string;
};

export type DataDeletedEvent = {
    data: boolean;
};
