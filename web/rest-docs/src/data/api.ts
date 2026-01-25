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

export const docs: Docs = docsData;
