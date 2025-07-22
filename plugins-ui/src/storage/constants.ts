export const storageKeys = {
  chain: "chain",
  currency: "currency",
  language: "language",
  theme: "theme",
  token: "token",
  vaultId: "vaultId",
} as const;

export type StorageKey = (typeof storageKeys)[keyof typeof storageKeys];
