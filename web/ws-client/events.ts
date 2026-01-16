export type EventKind = keyof APIEvents;
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
