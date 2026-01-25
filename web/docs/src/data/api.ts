import docsData from "../../../../api_docs.json";

export type Docs = typeof docsData;

type ApiDataMethods = Docs["methods"];
type ApiDataEvents = Docs["events"];
type ApiDataTypes = Docs["types"];

export type ItemType = "method" | "event" | "type";
export type MethodKeys = keyof ApiDataMethods;
export type EventKeys = keyof ApiDataEvents;
export type TypeKeys = keyof ApiDataTypes;

export type EventData = ApiDataEvents[EventKeys];
export type MethodData = ApiDataMethods[MethodKeys];
export type TypeData = ApiDataTypes[TypeKeys];

type TypeDataWithFields = Extract<TypeData, { fields: unknown }>;
export type FieldMetadata = TypeDataWithFields["fields"][number];

export type ErrorData = ApiDataMethods[MethodKeys]["errors"][number];
export type ExampleData =
    | ApiDataMethods[MethodKeys]["examples"][number]
    | ApiDataEvents[EventKeys]["examples"][number];

export function getTypeJson(typeName: TypeKeys | "null") {
    if (typeName === "null") return null;
    const type = docs.types[typeName];
    if ("jsonRepresentation" in type && type.jsonRepresentation) {
        return type.jsonRepresentation;
    }
    return null;
}

export const docs: Docs = docsData;
