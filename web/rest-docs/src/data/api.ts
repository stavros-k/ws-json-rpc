import docsData from "../../../../test.json";

export type Docs = typeof docsData;

type APIRoutes = Docs["routes"];
type ApiDataTypes = Docs["types"];

export type Routes = keyof APIRoutes;
export type RouteData = APIRoutes[Routes];
export type Verbs = keyof RouteData["verbs"];
export type VerbData = RouteData["verbs"][Verbs];
export type OperationID = VerbData["operationID"];

export type TypeKeys = keyof ApiDataTypes;
export type TypeData = ApiDataTypes[TypeKeys];

type TypeDataWithFields = Extract<TypeData, { fields: unknown[] }>;
export type FieldMetadata = NonNullable<TypeDataWithFields["fields"]>[number];

export type ItemType = "type";

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

export const docs: Docs = docsData;
