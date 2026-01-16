export type EventKind = keyof ApiEvents;
export type ApiEvents = {
    "data.created": { data: SomeEvent };
};

// Result for the SomeEvent method
export type SomeEvent = {
    /** The unique identifier for the result */
    id: string;
};
