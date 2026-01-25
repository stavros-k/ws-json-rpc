import docsData from "../../../../test.json";

export type Docs = typeof docsData;

type APIRoutes = Docs["routes"];
type ApiDataTypes = Docs["types"];

export type Routes = keyof APIRoutes;
export type Verbs = keyof APIRoutes[Routes]["verbs"];
// FIXME: this should be a union of all operationIDs
export type OperationIDs = APIRoutes[Routes]["verbs"][Verbs]["operationID"];

export type TypeKeys = keyof ApiDataTypes;
export type TypeData = ApiDataTypes[TypeKeys];

type TypeDataWithFields = Extract<TypeData, { fields: unknown[] }>;
export type FieldMetadata = NonNullable<TypeDataWithFields["fields"]>[number];

// Temporary: Map methods and events to empty objects since test.json uses routes
// TODO: Refactor components to use routes instead of methods/events
export type ItemType = "method" | "event" | "type";
export type MethodKeys = never; // No methods in REST API
export type EventKeys = never; // No events in REST API
export type ErrorData = never; // TODO: Map from route responses
export type ExampleData = never; // TODO: Map from route examples

export function getTypeJson(typeName: TypeKeys | "null") {
    if (typeName === "null") return null;
    const type = docs.types[typeName];
    if ("representations" in type && type.representations) {
        // Use jsonSchema if json is empty, otherwise use json
        const jsonRep = type.representations.json;
        if (jsonRep && jsonRep.trim() !== "") {
            return jsonRep;
        }
        // Fallback to jsonSchema if available
        const jsonSchema = type.representations.jsonSchema;
        if (jsonSchema && jsonSchema.trim() !== "") {
            return jsonSchema;
        }
    }
    return null;
}

// Extend docs with empty methods and events for backward compatibility
// TODO: Migrate all code to use routes instead
export const docs = {
    ...docsData,
    methods: {},
    events: {},
} as const;
