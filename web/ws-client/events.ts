export type EventKind = keyof APIEvents;
export type APIEvents = {
    "data.created": { data: DataCreatedEvent };
    "data.deleted": { data: DataDeletedEvent };
};

export type DataCreatedEvent = {
    data: string;
};

export type DataDeletedEvent = {
    data: boolean;
};
