// This file is generated. Do not edit manually.

export type MethodKind = typeof MethodKindValuesValues[keyof typeof MethodKindValuesValues];

export const MethodKindValuesValues = {
  MethodKindEcho: "echo",
  MethodKindAdd: "add",
  MethodKindDouble: "double",
  MethodKindPing: "ping",
  MethodKindSubscribe: "subscribe",
  MethodKindUnsubscribe: "unsubscribe"
} as const;
/**
* Status represents the status of a ping response

 * `StatusOK`
 * Sent when the ping is successful

 * `StatusNotFound`
 * This is the default status, so it is not necessary to specify it

 * `StatusError`
 * Sent when there is an error processing the ping
 */
export type Status = typeof StatusValuesValues[keyof typeof StatusValuesValues];

export const StatusValuesValues = {
  /** Sent when the ping is successful */
  StatusOK: "OK",
  /** This is the default status, so it is not necessary to specify it */
  StatusNotFound: "NotFound",
  /** Sent when there is an error processing the ping */
  StatusError: "Error"
} as const;
export type EventKind = typeof EventKindValuesValues[keyof typeof EventKindValuesValues];

export const EventKindValuesValues = {
  EventKindUserUpdate: "user.update",
  EventKindUserLogin: "user.login",
  EventKindDataProcessed: "data.processed"
} as const;
