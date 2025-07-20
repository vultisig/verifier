export const cssColorProperties = [
  "backgroundColor",
  "borderColor",
  "color",
  "fill",
] as const;

export type CSSColorProperties = (typeof cssColorProperties)[number];
