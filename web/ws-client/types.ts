import type { APIEvents, EventKind } from "./events";
import type { APIMethods, MethodKind } from "./methods";

type UUID = string;

// Incoming message is either a response or an event
export type IncomingMessage = ResponseMessage | EventMessage;

// Event handler function type
export type EventHandler<T> = (data: T) => void;

// Request message sent to the server
export type RequestMessage = {
    jsonrpc: "2.0";
    id: UUID;
    method: MethodKind;
    params: APIMethods[MethodKind]["req"];
};

// Response message with either result or error
export type ResponseMessage<Result = unknown> = {
    jsonrpc: "2.0";
    id: UUID;
} & (
    | { result: Result; error?: never }
    | {
          result?: never;
          error: { code: number; message: string; data?: unknown };
      }
);
export type EventMessage = {
    [K in EventKind]: {
        event: K;
        data: APIEvents[K];
    };
}[EventKind];
