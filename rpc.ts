// From: ws-json-rpc/internal/handlers.add.go:12
export type AddParams = {
  a: number;
  b: number;
};

// From: ws-json-rpc/internal/handlers.add.go:17
export type AddResult = {
  result: number;
  time: JSONTime;
};

// From: ws-json-rpc/internal/handlers.double.go:9
// Some other comment
export type DoubleParams = {
  value: number;
  other: number;
};

// From: ws-json-rpc/internal/handlers.double.go:14
export type DoubleResult = {
  result: number;
};

// From: ws-json-rpc/internal/handlers.echo.go:8
export type EchoParams = {
  message: string;
};

// From: ws-json-rpc/internal/handlers.echo.go:12
export type EchoResult = {
  echo: string;
};

// From: ws-json-rpc/internal/handlers.handlers.go:20
export type HandlerError = {
};

// From: ws-json-rpc/internal/handlers.handlers.go:12
// -32768 to -32000	Reserved - Do not use (reserved for pre-defined errors)
// -31999 to -1	Recommended for general application errors
// 1 to 999	Recommended for validation and input errors
// 1000 to 4999	Recommended for business logic errors
// 5000+	Recommended for system or infrastructure errors
export type HandlerErrorCode = 
  | -1
  | -2
  | -3;

// From: ws-json-rpc/internal/handlers.handlers.go:12
// -32768 to -32000	Reserved - Do not use (reserved for pre-defined errors)
// -31999 to -1	Recommended for general application errors
// 1 to 999	Recommended for validation and input errors
// 1000 to 4999	Recommended for business logic errors
// 5000+	Recommended for system or infrastructure errors
export const HandlerErrorCodeValues = {
  HandlerErrorCodeNotImplemented: -1,
  HandlerErrorCodeNotFound: -2,
  HandlerErrorCodeInternal: -3,
} as const;

// From: ws-json-rpc/internal/handlers.handlers.go:33
export type Handlers = {
};

// From: ws-json-rpc/internal/handlers.add.go:9
export type JSONTime = {
  ...Time;
};

// From: ws-json-rpc/internal/handlers.ping.go:20
export type PingResult = {
  message: string;
  status: Status;
};

// From: ws-json-rpc/internal/handlers.ping.go:9
// Status represents the status of a ping response
export type Status = 
  // Sent when the ping is successful
  | "OK"
  // This is the default status, so it is not necessary to specify it
  | "NotFound"
  // Sent when there is an error processing the ping
  | "Error";

// From: ws-json-rpc/internal/handlers.ping.go:9
// Status represents the status of a ping response
export const StatusValues = {
  // Sent when the ping is successful
  StatusOK: "OK",
  // This is the default status, so it is not necessary to specify it
  StatusNotFound: "NotFound",
  // Sent when there is an error processing the ping
  StatusError: "Error",
} as const;

// From: ws-json-rpc/internal/handlers.subscriptions.go:26
export type SubscribeParams = {
  event: EventKind;
};

// From: ws-json-rpc/internal/handlers.subscriptions.go:29
export type SubscribeResult = {
  subscribed: boolean;
};

// From: ws-json-rpc/internal/handlers.subscriptions.go:9
export type UnsubscribeParams = {
  event: EventKind;
};

// From: ws-json-rpc/internal/handlers.subscriptions.go:12
export type UnsubscribeResult = {
  unsubscribed: boolean;
};

// From: ws-json-rpc/internal/handlers.handlers.go:46
export type UserLoginEventResponse = {
  id: string;
  name: string;
};

// From: ws-json-rpc/internal/handlers.handlers.go:41
export type UserUpdateEventResponse = {
  id: string;
  name: string;
};


