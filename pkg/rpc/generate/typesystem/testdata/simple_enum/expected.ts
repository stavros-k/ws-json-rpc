// Available colors
export const Color = {
  /** Red color */
  Red: "red",
  /** Green color */
  Green: "green",
  /** Blue color */
  Blue: "blue",
} as const;

export type Color = typeof Color[keyof typeof Color];

export function isColor(value: unknown): value is Color {
	switch (value) {
		case "red":
		case "green":
		case "blue":
			return true;
		default:
			return false;
	}
}
