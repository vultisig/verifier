export const routeKeys = [
  "notFound",
  "plugins",
  "pluginDetails",
  "root",
] as const;

export type RouteKey = (typeof routeKeys)[number];

export const routeTree = {
  notFound: { path: "*" },
  plugins: { path: "/plugins" },
  pluginDetails: {
    link: (id: string | number) => `/plugins/${id}`,
    path: "/plugins/:id",
  },
  root: { path: "/" },
} satisfies Record<
  RouteKey,
  { path: string; link?: (...args: (string | number)[]) => string }
>;
