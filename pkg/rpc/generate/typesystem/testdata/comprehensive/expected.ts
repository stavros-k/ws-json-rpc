// User status
export const Status = {
  /** User is active */
  Active: "active",
  /** User is inactive */
  Inactive: "inactive",
} as const;

export type Status = typeof Status[keyof typeof Status];

export function isStatus(value: unknown): value is Status {
	switch (value) {
		case "active":
		case "inactive":
			return true;
		default:
			return false;
	}
}

// A string map
export type StringMap = Record<string, string>;

// A list of tags
export type Tags = Array<string>;

// User entity
export type User = {
  /** When the user was created */
  createdAt: string;
  /** User ID */
  id: UserID;
  /** User status */
  status: Status;
  /** User tags */
  tags: Array<string>;
};

// Unique identifier for a user
export type UserID = string;

