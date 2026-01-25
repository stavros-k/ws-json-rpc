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

export type ItemType = "type" | "operation";

// Operation types
export type OperationData = VerbData & {
    route: Routes;
    verb: string;
};

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

// Get all operations from routes
export function getAllOperations(): OperationData[] {
    const operations: OperationData[] = [];

    for (const [route, routeData] of Object.entries(docs.routes)) {
        for (const [verb, verbData] of Object.entries(routeData.verbs)) {
            operations.push({
                ...verbData,
                route: route as Routes,
                verb: verb,
            });
        }
    }

    return operations;
}

export const docs: Docs = docsData;
