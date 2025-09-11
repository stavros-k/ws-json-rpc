export type Methods = {
  [method: string]: {
    req: any;
    res?: any; // Optional res means it's a notification
  };
};
export type Events = {
  [event: string]: any;
};
