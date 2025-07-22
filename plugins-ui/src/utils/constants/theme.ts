export const themes = ["default"] as const;

export type Theme = (typeof themes)[number];

export const defaultTheme: Theme = "default";
